package http

// convertGenderToFrontend convierte el género de BD (M/F/O) a frontend (Masculino/Femenino/Otro)
func convertGenderToFrontend(gender string) string {
	switch gender {
	case "M":
		return "Masculino"
	case "F":
		return "Femenino"
	case "O":
		return "Otro"
	default:
		return gender // Devolver tal cual si no coincide
	}
}

// convertGenderToDatabase convierte el género de frontend (Masculino/Femenino/Otro) a BD (M/F/O)
func convertGenderToDatabase(gender string) string {
	switch gender {
	case "Masculino":
		return "M"
	case "Femenino":
		return "F"
	case "Otro":
		return "O"
	default:
		return gender // Devolver tal cual si no coincide
	}
}
