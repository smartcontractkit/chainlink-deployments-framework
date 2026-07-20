package timelockdelay

import "errors"

// ErrUnsetTimelockDelayUnverified indicates a schedule proposal has no delay set and
// on-chain minDelay could not be read for every timelock chain.
var ErrUnsetTimelockDelayUnverified = errors.New("timelock delay unset and unable to resolve from on-chain minDelay")
