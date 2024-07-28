package client

type Image struct {
	Id                  string                 `json:"id"`
	FileName            string                 `json:"fileName"`
	ComputedFileName    string                 `json:"computedFileName"`
	ExifData            map[string]interface{} `json:"exifData"`
	CapturedAt          DateTime               `json:"capturedAt"`
	CapturedAtCorrected DateTime               `json:"capturedAtCorrected"`
	User                string                 `json:"user"`
	ImageTagAssignments []string               `json:"imageTagAssignments"`
	Upload              string                 `json:"upload"`
	Project             string                 `json:"project"`
	Size                int64                  `json:"size"`
	StorageId           string                 `json:"storageId"`
	DownloadUrls        map[string]string      `json:"downloadUrls"`
	Expand              *ImageExpand           `json:"expand"`
	CreatedAt           DateTime               `json:"createdAt"`
	UpdatedAt           DateTime               `json:"updatedAt"`
}

type ImageExpand struct {
	User                        *User                `json:"user"`
	Upload                      *Upload              `json:"upload"`
	Project                     *Project             `json:"project"`
	ImageTagAssignmentsViaImage []ImageTagAssignment `json:"image_tag_assignments_via_image"`
}

type ImageTagAssignment struct {
	Id       string                    `json:"id"`
	Type     string                    `json:"type"`
	ImageTag string                    `json:"imageTag"`
	Image    string                    `json:"image"`
	Expand   *ImageTagAssignmentExpand `json:"expand"`
}

type ImageTagAssignmentExpand struct {
	ImageTag *ImageTag `json:"imageTag"`
	Image    *Image    `json:"image"`
}

type ImageTag struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAlbum     bool   `json:"isAlbum"`
	Type        string `json:"type"`
	Project     string `json:"project"`
}

type User struct {
	Id           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Verified     bool   `json:"verified"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	CopyrightTag string `json:"copyRightTag"`
	Active       bool   `json:"active"`
	Role         string `json:"role"`
}

type Upload struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Copyright          string `json:"copyright"`
	CopyrightReference string `json:"copyRightReference"`
	LocationName       string `json:"locationName"`
	LocationCode       string `json:"locationCode"`
	LocationCity       string `json:"locationCity"`
}

type AuthWithPasswordResponse struct {
	Record struct {
		Active             bool     `json:"active"`
		ActiveProject      string   `json:"activeProject"`
		Avatar             string   `json:"avatar"`
		CollectionId       string   `json:"collectionId"`
		CollectionName     string   `json:"collectionName"`
		CopyrightTag       string   `json:"copyrightTag"`
		Created            string   `json:"created"`
		Email              string   `json:"email"`
		EmailVisibility    bool     `json:"emailVisibility"`
		FirstName          string   `json:"firstName"`
		Id                 string   `json:"id"`
		LastName           string   `json:"lastName"`
		ProjectAssignments []string `json:"projectAssignments"`
		Role               string   `json:"role"`
		Updated            string   `json:"updated"`
		Username           string   `json:"username"`
		Verified           bool     `json:"verified"`
	} `json:"record"`
	Token string `json:"token"`
}
