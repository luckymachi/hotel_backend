package application

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator contiene funciones de validación de datos
type Validator struct{}

// ValidateEmail valida el formato de un email
func (v *Validator) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("el email es requerido")
	}

	// Regex básico para email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("el formato del email '%s' no es válido", email)
	}

	return nil
}

// ValidatePhone valida el formato de un teléfono
func (v *Validator) ValidatePhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("el teléfono es requerido")
	}

	// Limpiar espacios y guiones
	cleanPhone := strings.ReplaceAll(phone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")

	// Verificar que solo contenga dígitos y opcionalmente un +
	phoneRegex := regexp.MustCompile(`^\+?\d{7,15}$`)

	if !phoneRegex.MatchString(cleanPhone) {
		return fmt.Errorf("el teléfono '%s' debe tener entre 7 y 15 dígitos", phone)
	}

	return nil
}

// ValidateDocumentNumber valida el número de documento
func (v *Validator) ValidateDocumentNumber(docNumber string) error {
	if docNumber == "" {
		return fmt.Errorf("el número de documento es requerido")
	}

	// Limpiar espacios y guiones
	cleanDoc := strings.ReplaceAll(docNumber, " ", "")
	cleanDoc = strings.ReplaceAll(cleanDoc, "-", "")

	// Debe tener entre 6 y 15 caracteres alfanuméricos
	if len(cleanDoc) < 6 || len(cleanDoc) > 15 {
		return fmt.Errorf("el número de documento debe tener entre 6 y 15 caracteres")
	}

	// Verificar que solo contenga letras y números
	docRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	if !docRegex.MatchString(cleanDoc) {
		return fmt.Errorf("el número de documento solo puede contener letras y números")
	}

	return nil
}

// ValidateName valida que un nombre no esté vacío y tenga formato válido
func (v *Validator) ValidateName(name, fieldName string) error {
	if name == "" {
		return fmt.Errorf("el %s es requerido", fieldName)
	}

	name = strings.TrimSpace(name)

	if len(name) < 2 {
		return fmt.Errorf("el %s debe tener al menos 2 caracteres", fieldName)
	}

	if len(name) > 50 {
		return fmt.Errorf("el %s no puede tener más de 50 caracteres", fieldName)
	}

	// Solo letras, espacios, acentos y algunos caracteres especiales
	nameRegex := regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚñÑ\s\-']+$`)

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("el %s contiene caracteres no válidos", fieldName)
	}

	return nil
}

// ValidateGender valida que el género sea M o F
func (v *Validator) ValidateGender(gender string) error {
	gender = strings.ToUpper(strings.TrimSpace(gender))

	if gender != "M" && gender != "F" {
		return fmt.Errorf("el género debe ser 'M' (masculino) o 'F' (femenino)")
	}

	return nil
}

// ValidatePersonalData valida todos los datos personales
func (v *Validator) ValidatePersonalData(
	nombre, primerApellido, segundoApellido, numeroDocumento, genero, email, telefono1, telefono2 string,
) []error {
	var errors []error

	// Validar nombre
	if err := v.ValidateName(nombre, "nombre"); err != nil {
		errors = append(errors, err)
	}

	// Validar primer apellido
	if err := v.ValidateName(primerApellido, "primer apellido"); err != nil {
		errors = append(errors, err)
	}

	// Validar segundo apellido (opcional)
	if segundoApellido != "" {
		if err := v.ValidateName(segundoApellido, "segundo apellido"); err != nil {
			errors = append(errors, err)
		}
	}

	// Validar documento
	if err := v.ValidateDocumentNumber(numeroDocumento); err != nil {
		errors = append(errors, err)
	}

	// Validar género
	if err := v.ValidateGender(genero); err != nil {
		errors = append(errors, err)
	}

	// Validar email
	if err := v.ValidateEmail(email); err != nil {
		errors = append(errors, err)
	}

	// Validar teléfono principal
	if err := v.ValidatePhone(telefono1); err != nil {
		errors = append(errors, err)
	}

	// Validar teléfono secundario (opcional)
	if telefono2 != "" {
		if err := v.ValidatePhone(telefono2); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// FormatValidationErrors formatea una lista de errores en un mensaje legible
func (v *Validator) FormatValidationErrors(errors []error) string {
	if len(errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("❌ Se encontraron los siguientes errores en los datos proporcionados:\n\n")

	for i, err := range errors {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
	}

	sb.WriteString("\nPor favor, corrige estos datos y vuelve a intentarlo.")

	return sb.String()
}
