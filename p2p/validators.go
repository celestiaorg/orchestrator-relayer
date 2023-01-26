package p2p

// ValsetConfirmValidator runs stateless checks on valset confirms when submitting them to the DHT.
type ValsetConfirmValidator struct{}

func (vcv ValsetConfirmValidator) Validate(key string, value []byte) error {
	// TODO Should verify that the valset confirm is valid, i.e. running stateless checks on it.
	// The checks should include:
	// - Correct signature verification
	// - Correct fields checks. Example, checking if an address field as a correctly formatted address.
	return nil
}

func (vcv ValsetConfirmValidator) Select(key string, values [][]byte) (int, error) {
	// TODO Should run the same stateless checks as the `Validate` function to avoid querying
	// faulty values.
	return 0, nil
}

// DataCommitmentConfirmValidator runs stateless checks on data commitment confirms when submitting to the DHT.
type DataCommitmentConfirmValidator struct{}

func (dcv DataCommitmentConfirmValidator) Validate(key string, value []byte) error {
	// TODO Should verify that the data commitment confirm is valid, i.e. running stateless checks on it.
	// The checks should include:
	// - Correct signature verification
	// - Correct fields checks. Example, checking if an address field as a correctly formatted address.
	return nil
}

func (dcv DataCommitmentConfirmValidator) Select(key string, values [][]byte) (int, error) {
	// TODO Should run the same stateless checks as the `Validate` function to avoid querying
	// faulty values.
	return 0, nil
}
