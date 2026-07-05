package model

// TrackDownloadInfo represents the XML payload used to build signed MP3 download URLs.
type TrackDownloadInfo struct {
	// Host is the CDN host serving the MP3 file.
	Host string `xml:"host"`
	// Path is the MP3 file path on the CDN host.
	Path string `xml:"path"`
	// Ts is the request timestamp token used in URL signing.
	Ts string `xml:"ts"`
	// Region is the CDN region identifier.
	Region int `xml:"region"`
	// S is the signing salt token from the download-info response.
	S string `xml:"s"`
}
