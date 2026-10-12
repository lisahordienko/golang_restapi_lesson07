package handler

type BookRequest struct {
	Title      string `json:"title"`
	ISBN       string `json:"isbn"`
	CategoryID uint   `json:"category_id"`
	AuthorIDs  []uint `json:"author_ids"`
}
