package assistant

// Delivery is how an assistant gets Quiver's always-on skills
// (docs/spike-hooks.md).
type Delivery int

const (
	// DeliveryHook: a session-start hook adds the always-on skills' text at
	// the start of a session.
	DeliveryHook Delivery = iota
	// DeliveryInstructions: the model gets Quiver's catalog up front and
	// loads the always-on skills itself.
	DeliveryInstructions
	// DeliveryOnRequest: skills load only when the prompt asks for them.
	DeliveryOnRequest
)

// HasHook reports whether the assistant gets always-on skills from a hook, so
// they always reach the model.
func (d Delivery) HasHook() bool { return d == DeliveryHook }

// Level is a short name for the level, for status and JSON.
func (d Delivery) Level() string {
	switch d {
	case DeliveryHook:
		return "hook"
	case DeliveryInstructions:
		return "instructions"
	default:
		return "on request"
	}
}

// Describe is a sentence about how the assistant gets always-on skills.
func (d Delivery) Describe() string {
	switch d {
	case DeliveryHook:
		return "always-on skills load from a session-start hook."
	case DeliveryInstructions:
		return "no session-start hook. Quiver's skill list is sent up front, and the model loads the always-on skills itself."
	default:
		return "no session-start hook. Quiver's skills are loaded when the model decides to."
	}
}
