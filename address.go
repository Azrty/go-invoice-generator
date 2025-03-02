package generator

// Address represent an address
type Address struct {
	Address    string `json:"address,omitempty" validate:"required"`
	Address2   string `json:"address_2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	State       string `json:"State,omitempty"`
	Country    string `json:"country,omitempty"`
}

// ToString output address as string
// Line break are added for new lines
func (a *Address) ToString() string {
	addrString := a.Address

	if len(a.Address2) > 0 {
		addrString += "\n"
		addrString += a.Address2
	}
	if len(a.City) > 0 {
		addrString += "\n"
		addrString += a.City
	} else {
		addrString += "\n"
	}
	if len(a.State) > 0 {
		addrString += ", "
		addrString += a.State
	}

	if len(a.PostalCode) > 0 {
		addrString += " "
		addrString += a.PostalCode
	}

	if len(a.Country) > 0 {
		addrString += "\n"
		addrString += a.Country
	}

	return addrString
}
