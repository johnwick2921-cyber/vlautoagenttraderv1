// Package branding supplies display names only. These are never identifiers.
package branding

import _ "embed"

// The UI and Vite read these same files. Keep operational names out of them.
//
//go:embed product.txt
var productName string

//go:embed persona.txt
var personaName string

func ProductName() string { return productName }
func PersonaName() string { return personaName }
