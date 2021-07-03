package tile

import (
	"fmt"
	"strconv"
)

const (
	// Property types
	// see doc.mapeditor.org/en/stable/reference/tmx-map-format/#properties
	PropString = "string"
	PropInt    = "int"
	PropBool   = "bool"
)

// Properties is a more straight forward []*Property (used by the raw XML)
// that handles types a bit more gracefully.
type Properties struct {
	ints    map[string]int
	strings map[string]string
	bools   map[string]bool
}

// NewProperties returns an empty properties
func NewProperties() *Properties {
	return &Properties{
		ints:    map[string]int{},
		strings: map[string]string{},
		bools:   map[string]bool{},
	}
}

// Merge properties `o` into this properties
func (p *Properties) Merge(o *Properties) *Properties {
	for k, v := range o.ints {
		p.ints[k] = v
	}
	for k, v := range o.strings {
		p.strings[k] = v
	}
	for k, v := range o.bools {
		p.bools[k] = v
	}
	return p
}

// toList mutates our nicer properties wrapper back into []*Property understood
// by the XML encoder
func (p *Properties) toList() []*Property {
	ps := []*Property{}
	for k, v := range p.ints {
		ps = append(ps, &Property{
			Name:  k,
			Value: fmt.Sprintf("%d", v),
			Type:  PropInt,
		})
	}
	for k, v := range p.bools {
		ps = append(ps, &Property{
			Name:  k,
			Value: fmt.Sprintf("%v", v),
			Type:  PropBool,
		})
	}
	for k, v := range p.strings {
		ps = append(ps, &Property{
			Name:  k,
			Value: v,
			Type:  PropString,
		})
	}
	return ps
}

// newPropertiesFromList turns the XML []Property into our nicer properties
// wrapper struct.
func newPropertiesFromList(in []*Property) *Properties {
	ps := &Properties{
		ints:    map[string]int{},
		strings: map[string]string{},
		bools:   map[string]bool{},
	}

	for _, i := range in {
		switch i.Type {
		case "int":
			v, _ := strconv.ParseInt(i.Value, 10, 64)
			ps.SetInt(i.Name, int(v))
		case "bool":
			ps.SetBool(i.Name, i.Value == "true")
		default:
			// we don't use float, image etc
			ps.SetString(i.Name, i.Value)
		}
	}

	return ps
}

func (p *Properties) String(key string) (string, bool) {
	v, ok := p.strings[key]
	return v, ok
}

func (p *Properties) SetString(key, value string) {
	p.strings[key] = value
	delete(p.ints, key)
	delete(p.bools, key)
}

func (p *Properties) Int(key string) (int, bool) {
	v, ok := p.ints[key]
	return v, ok
}

func (p *Properties) SetInt(key string, value int) {
	p.ints[key] = value
	delete(p.strings, key)
	delete(p.bools, key)
}

func (p *Properties) Bool(key string) (bool, bool) {
	v, ok := p.bools[key]
	return v, ok
}

func (p *Properties) SetBool(key string, value bool) {
	p.bools[key] = value
	delete(p.strings, key)
	delete(p.ints, key)
}
