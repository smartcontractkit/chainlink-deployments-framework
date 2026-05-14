package mcms

// MCMSRole represents a named role within the MCMS system (e.g. proposer, bypasser, canceller).
type MCMSRole string //nolint:revive // renaming would be a breaking change

const (
	ProposerRole  MCMSRole = "PROPOSER"
	BypasserRole  MCMSRole = "BYPASSER"
	CancellerRole MCMSRole = "CANCELLER"
)

func (role MCMSRole) String() string {
	return string(role)
}
