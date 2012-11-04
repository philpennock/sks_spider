package sks_spider

type CountrySet []string

func NewCountrySet(s string) *CountrySet {
	return &CountrySet{""}
}

func (cs *CountrySet) HasCountry(s string) bool {
	return false
}

func (cs *CountrySet) String() string {
	return "not-implemented"
}
