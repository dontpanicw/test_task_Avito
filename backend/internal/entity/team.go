package entity

// Team представляет команду с участниками
type Team struct {
	TeamName string
	Members  []TeamMember
}

// TeamMember представляет участника команды
type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

