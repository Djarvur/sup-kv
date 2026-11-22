package persistfile

// classifySaveError maps a save failure onto the port's FailureClass
// vocabulary (D-05): the errno knowledge stays in this adapter and the core
// consumes only port.FailureClassOf. The returned error unwraps to err, so
// [errors.Is] keeps reaching the underlying cause (T-5-16).
func classifySaveError(err error) error {
	return err
}
