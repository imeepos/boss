package workerapi

type workerSettingsReq struct {
	Online      bool     `json:"online"`
	RadiusKm    int16    `json:"radiusKm"`
	AcceptTypes []string `json:"acceptTypes"`
}
