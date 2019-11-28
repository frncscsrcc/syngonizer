package kubectl

// PodMetadata ...
type PodMetadata struct {
	Name      string    `json:"name"`
	Labels    PodLabels `json:"labels"`
	Namespace string    `json:"namespace"`
}

// PodLabels ...
type PodLabels struct {
	App string `json:"app"`
}

// PodItem ...
type PodItem struct {
	Metadata PodMetadata `json:"metadata"`
}

// PodList ...
type PodList struct {
	PodItems []PodItem `json:"items"`
}
