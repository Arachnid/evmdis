package evmdis

type EvmState interface {
	Advance() ([]EvmState, error)
}

func ExecuteAbstractly(initial EvmState) error {
	stack := []EvmState{initial}
	seen := make(map[EvmState]bool)

	for len(stack) > 0 {
		var state EvmState
		state, stack = stack[len(stack)-1], stack[:len(stack)-1]
		nextStates, err := state.Advance()
		if err != nil {
			return err
		}
		for _, nextState := range nextStates {
			if !seen[nextState] {
				stack = append(stack, nextState)
				seen[nextState] = true
			}
		}
	}

	return nil
}
