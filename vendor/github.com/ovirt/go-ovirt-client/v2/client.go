package ovirtclient

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net/http"
	"sync"
	"time"

	ovirtsdk4 "github.com/ovirt/go-ovirt"
)

// Client is a simplified client for the oVirt API.
//
//goland:noinspection GoDeprecation
type Client interface {
	// GetURL returns the oVirt engine base URL.
	GetURL() string
	// Reconnect triggers the client to reauthenticate against the oVirt Engine.
	Reconnect() (err error)
	// WithContext creates a subclient with the specified context applied.
	WithContext(ctx context.Context) Client
	// GetContext returns the current context of the client. May be nil.
	GetContext() context.Context

	AffinityGroupClient
	DiskClient
	DiskAttachmentClient
	VMClient
	NICClient
	VNICProfileClient
	NetworkClient
	DatacenterClient
	ClusterClient
	StorageDomainClient
	HostClient
	TemplateClient
	TemplateDiskClient
	TestConnectionClient
	TagClient
	FeatureClient
	InstanceTypeClient
	GraphicsConsoleClient
}

// ClientWithLegacySupport is an extension of Client that also offers the ability to retrieve the underlying
// SDK connection or a configured HTTP client.
type ClientWithLegacySupport interface {
	// GetSDKClient returns a configured oVirt SDK client for the use cases that are not covered by goVirt.
	GetSDKClient() *ovirtsdk4.Connection

	// GetHTTPClient returns a configured HTTP client for the oVirt engine. This can be used to send manual
	// HTTP requests to the oVirt engine.
	GetHTTPClient() http.Client

	Client
}

type oVirtClient struct {
	reconnectLock   *sync.Mutex
	conn            *ovirtsdk4.Connection
	ctx             context.Context
	httpClient      http.Client
	logger          Logger
	url             string
	username        string
	password        string
	tlsConfig       *tls.Config
	extraSettings   ExtraSettings
	nonSecureRandom *rand.Rand
	verify          func(connection Client) error
}

func (o *oVirtClient) WithContext(ctx context.Context) Client {
	return &oVirtClient{
		o.reconnectLock,
		o.conn,
		ctx,
		o.httpClient,
		o.logger.WithContext(ctx),
		o.url,
		o.username,
		o.password,
		o.tlsConfig,
		o.extraSettings,
		o.nonSecureRandom,
		o.verify,
	}
}

func (o *oVirtClient) GetContext() context.Context {
	return o.ctx
}

// ReconnectPolicy configures the SDK's acquisition interval or JWT percentage.
type ReconnectPolicy = ovirtsdk4.ReconnectPolicy

// ReconnectIfDue uses the same login and publication path as Reconnect. With no
// policy it reconnects at 90 percent of the cached JWT lifetime.
func (o *oVirtClient) ReconnectIfDue(policies ...ReconnectPolicy) (bool, error) {
	return o.reconnect(true, policies...)
}

func (o *oVirtClient) Reconnect() error {
	_, err := o.reconnect(false)
	return err
}

func (o *oVirtClient) reconnect(onlyIfDue bool, policies ...ReconnectPolicy) (bool, error) {
	policy := ReconnectPolicy{LifetimePercentage: 90}
	if onlyIfDue {
		if len(policies) > 1 {
			return false, newError(EBadArgument, "only one reconnect policy may be supplied")
		}
		if len(policies) == 1 {
			policy = policies[0]
		}
		if err := policy.Validate(); err != nil {
			o.logger.Warningf("Rejecting invalid scheduled reconnect policy")
			return false, err
		}
	}
	o.reconnectLock.Lock()
	defer o.reconnectLock.Unlock()
	if onlyIfDue {
		if o.conn == nil {
			return false, newError(EBug, "scheduled reconnect requires an initialized SDK connection")
		}
		due, err := o.conn.ReconnectDue(time.Now(), policy)
		if err != nil {
			o.logger.Warningf("Reconnect deadline is unavailable; retaining active authentication")
			return false, err
		}
		if !due {
			return false, nil
		}
	}

	o.logger.Debugf("Building a detached SDK connection with the configured settings")
	candidate, err := o.buildConnection()
	if err != nil {
		o.logger.Warningf("Could not build the reconnect candidate; retaining active authentication")
		return false, err
	}
	if o.conn == nil {
		// Construction remains lazy when verification is disabled. An engine
		// outage must not prevent the CSI driver from starting its probe service.
		if o.verify != nil {
			if err := o.verifyCandidate(candidate); err != nil {
				o.logger.Warningf("Initial SDK connection verification failed")
				return false, err
			}
		}
		o.conn = candidate
		o.logger.Debugf("Initialized the shared SDK connection")
		return true, nil
	}

	o.logger.Infof("Reconnecting to oVirt with a fresh credential login")
	if err := candidate.Authenticate(); err != nil {
		// Authentication responses may contain secrets; do not log err.
		o.logger.Warningf("Reconnect login failed; retaining active authentication")
		return false, err
	}
	o.logger.Debugf("Verifying the detached reconnect candidate before publication")
	if err := o.verifyCandidate(candidate); err != nil {
		o.logger.Warningf("Reconnect verification failed; retaining active authentication")
		return false, err
	}
	if onlyIfDue && policy.Interval == 0 {
		oldExpiry, err := o.conn.TokenExpiry()
		if err != nil {
			return false, err
		}
		newExpiry, err := candidate.TokenExpiry()
		if err != nil {
			o.logger.Warningf("Replacement JWT expiry is unavailable; retaining active authentication")
			return false, err
		}
		if newExpiry.IsZero() || !newExpiry.After(oldExpiry) {
			o.logger.Warningf("Replacement JWT does not extend expiry; retaining active authentication")
			return false, newError(EUnidentified, "replacement JWT does not extend the active expiry")
		}
	}
	if err := o.conn.ReplaceAuthentication(candidate); err != nil {
		o.logger.Warningf("Reconnect publication failed; retaining active authentication")
		return false, err
	}
	if expiry, err := o.conn.TokenExpiry(); err == nil && !expiry.IsZero() {
		o.logger.Infof("Reconnect completed; active JWT expires at %s", expiry.UTC().Format(time.RFC3339))
	} else {
		o.logger.Infof("Reconnect completed; active token has no known JWT expiry")
	}
	return true, nil
}

func (o *oVirtClient) buildConnection() (*ovirtsdk4.Connection, error) {
	connBuilder := ovirtsdk4.NewConnectionBuilder().
		URL(o.url).
		Username(o.username).
		Password(o.password).
		TLSConfig(o.tlsConfig)
	if err := processExtraSettings(o.extraSettings, connBuilder); err != nil {
		return nil, err
	}

	conn, err := connBuilder.Build()
	if err != nil {
		return nil, wrap(err, EUnidentified, "failed to create underlying oVirt connection")
	}
	return conn, nil
}

func (o *oVirtClient) verifyCandidate(candidate *ovirtsdk4.Connection) error {
	if o.verify == nil {
		return candidate.Test()
	}
	// Verification must use the candidate, never the active connection. Give
	// verification/recovery its own lock so it cannot reenter reconnectLock.
	verificationClient := *o
	verificationClient.conn = candidate
	verificationClient.reconnectLock = &sync.Mutex{}
	verificationClient.verify = nil
	return o.verify(&verificationClient)
}

func (o *oVirtClient) GetSDKClient() *ovirtsdk4.Connection {
	return o.conn
}

func (o *oVirtClient) GetHTTPClient() http.Client {
	return o.httpClient
}

func (o *oVirtClient) GetURL() string {
	return o.url
}
