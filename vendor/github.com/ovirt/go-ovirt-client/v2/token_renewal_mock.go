package ovirtclient

// ReconnectIfDue keeps mock clients compatible with probe-driven reconnects.
// Mock clients have no expiring authentication state and never need renewal.
func (m *mockClient) ReconnectIfDue(policies ...ReconnectPolicy) (bool, error) {
	if len(policies) > 1 {
		return false, newError(EBadArgument, "only one reconnect policy may be supplied")
	}
	if len(policies) == 1 {
		if err := policies[0].Validate(); err != nil {
			m.logger.Warningf("Rejecting invalid mock scheduled reconnect policy")
			return false, err
		}
	}
	m.logger.Debugf("Checking mock scheduled reconnect policy")
	m.logger.Debugf("Mock reconnect is not required")
	return false, nil
}
