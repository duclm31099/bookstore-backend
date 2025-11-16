package email

type VerificationEmailData struct {
	Email      string
	VerifyLink string
	ExpiresIn  string
}
type ResetPasswordData struct {
	Email     string
	Token     string
	ExpiresIn string
}
type EmailRequest struct {
	To          []string     // Recipients
	Cc          []string     // Carbon copy (optional)
	Bcc         []string     // Blind carbon copy (optional)
	Subject     string       // Email subject
	Body        string       // Email body (HTML or plain text)
	IsHTML      bool         // true for HTML, false for plain text
	Attachments []Attachment // File attachments (optional)
}
type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}
