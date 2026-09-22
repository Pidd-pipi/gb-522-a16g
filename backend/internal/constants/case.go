package constants

type CaseStatus string

const (
	CaseDraft         CaseStatus = "draft"
	CaseAnalyzing     CaseStatus = "analyzing"
	CasePendingReview CaseStatus = "pending_review"
	CaseConfirmed     CaseStatus = "confirmed"
	CaseClosed        CaseStatus = "closed"
)

var transitions = map[CaseStatus]map[CaseStatus]bool{
	CaseDraft:         {CaseAnalyzing: true},
	CaseAnalyzing:     {CaseDraft: true, CasePendingReview: true},
	CasePendingReview: {CaseConfirmed: true},
	CaseConfirmed:     {CaseClosed: true},
	CaseClosed:        {},
}

func (s CaseStatus) Valid() bool {
	_, ok := transitions[s]
	return ok
}

func CanTransition(from, to CaseStatus) bool { return transitions[from][to] }

const (
	RoleAnalyst  = "analyst"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
)

func ValidRole(role string) bool {
	return role == RoleAnalyst || role == RoleReviewer || role == RoleAdmin
}
