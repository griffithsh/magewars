package game

// Sprite Component is a renderable slice of a texture.
type Sprite struct {
	Texture string `json:"texture"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"w"`
	H       int    `json:"h"`
}
