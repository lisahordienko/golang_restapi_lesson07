package handler

type AuthorRequest struct {
	FullName string  `json:"full_name"`
	Bio      *string `json:"bio"`
}
