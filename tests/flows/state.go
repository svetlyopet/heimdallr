package flows

// FleetState holds IDs produced during the fleet seed/run phases.
type FleetState struct {
	RunID          string
	OrphanID       string
	SecondOrphanID string
	ServerID       string
	SecondServerID string
	Hostname       string
	SecondHostname string
}

// ComplianceState holds IDs produced during the compliance seed/run phases.
type ComplianceState struct {
	ApplicationID  string
	Token          string
	ReleaseID      string
	ReleaseVersion string
	ReportID       string
	CommitSHA      string
}

// OperationsState holds IDs for operations verification.
type OperationsState struct {
	AutomationID   string
	JobIDSuccess   string
	JobIDFailure   string
	AutomationName string
}
