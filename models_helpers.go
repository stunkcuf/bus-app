package main

// Helper methods for ECSEStudent to handle nullable fields in templates
func (e ECSEStudent) GetGrade() string {
	if e.Grade.Valid {
		return e.Grade.String
	}
	return ""
}

func (e ECSEStudent) GetIEPStatus() string {
	if e.IEPStatus.Valid {
		return e.IEPStatus.String
	}
	return ""
}

func (e ECSEStudent) GetPrimaryDisability() string {
	if e.PrimaryDisability.Valid {
		return e.PrimaryDisability.String
	}
	return ""
}

func (e ECSEStudent) GetServiceMinutes() int {
	if e.ServiceMinutes.Valid {
		return int(e.ServiceMinutes.Int32)
	}
	return 0
}

func (e ECSEStudent) IsTransportationRequired() bool {
	if e.TransportationRequired.Valid {
		return e.TransportationRequired.Bool
	}
	return false
}

func (e ECSEStudent) GetBusRoute() string {
	if e.BusRoute.Valid {
		return e.BusRoute.String
	}
	return ""
}

func (e ECSEStudent) GetEnrollmentStatus() string {
	if e.EnrollmentStatus.Valid {
		return e.EnrollmentStatus.String
	}
	return ""
}

func (e ECSEStudent) GetParentName() string {
	if e.ParentName.Valid {
		return e.ParentName.String
	}
	return ""
}

func (e ECSEStudent) GetParentPhone() string {
	if e.ParentPhone.Valid {
		return e.ParentPhone.String
	}
	return ""
}

func (e ECSEStudent) GetParentEmail() string {
	if e.ParentEmail.Valid {
		return e.ParentEmail.String
	}
	return ""
}

func (e ECSEStudent) GetCity() string {
	if e.City.Valid {
		return e.City.String
	}
	return ""
}

func (e ECSEStudent) GetState() string {
	if e.State.Valid {
		return e.State.String
	}
	return ""
}

func (e ECSEStudent) GetZipCode() string {
	if e.ZipCode.Valid {
		return e.ZipCode.String
	}
	return ""
}

func (e ECSEStudent) GetNotes() string {
	if e.Notes.Valid {
		return e.Notes.String
	}
	return ""
}

func (e ECSEStudent) GetDateOfBirth() string {
	if e.DateOfBirth.Valid {
		return e.DateOfBirth.String
	}
	return ""
}

func (e ECSEStudent) GetAddress() string {
	if e.Address.Valid {
		return e.Address.String
	}
	return ""
}