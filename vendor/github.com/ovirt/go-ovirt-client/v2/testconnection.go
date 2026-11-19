package ovirtclient

// TestConnectionClient defines the functions related to testing the connection.
type TestConnectionClient interface {
	// Test tests if the connection is alive or not.
	Test(retries ...RetryStrategy) error
}

func (o *oVirtClient) Test(retries ...RetryStrategy) error {
	if o.verifying {
		o.logger.Debugf("Checking candidate authorization without recursive authentication recovery")
		return o.conn.Test()
	}
	retries = defaultRetries(retries, defaultReadTimeouts(o))
	return retry(
		"testing oVirt engine connection",
		o.logger,
		retries,
		func() error {
			o.logger.Debugf("Checking API authorization with the active authentication")
			if err := o.conn.SystemService().Connection().Test(); err != nil {
				// Recovery strategies require an identified EngineError. Legacy
				// SSO expiry arrives as an HTML invalid_grant response from the SDK.
				o.logger.Debugf("Classifying connection failure for retry and authentication recovery")
				return wrap(err, EUnidentified, "failed to test oVirt engine connection")
			}
			return nil
		},
	)
}

func (m *mockClient) Test(retries ...RetryStrategy) error {
	retries = defaultRetries(retries, defaultReadTimeouts(m))
	return retry(
		"testing oVirt engine connection",
		nil,
		retries,
		func() error {
			return nil
		},
	)
}
