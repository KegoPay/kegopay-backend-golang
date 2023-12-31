package auth

import "kego.com/entities"

type ClaimsData struct {
    Issuer       string
    UserID       string
    FirstName    string
    LastName     string
    Email        *string
    Phone        *entities.PhoneNumber
    ExpiresAt    int64
    IssuedAt     int64
    UserAgent    string
    DeviceID     string
    AppVersion   string
}
