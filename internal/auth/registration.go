package auth

type Registration struct {
	Code     string `json:"code" firestore:"Code"`
	Status   string `json:"status" firestore:"Status"`
	UserName string `json:"username" firestore:"UserName"`
}
