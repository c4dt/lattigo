package ckks

import (
	"errors"
	"github.com/ldsec/lattigo/ring"
)

// Operand is an interface allowing operations between
// Plaintext and Ciphertext types.
type Operand interface {
	Element() *ckksElement
	Degree() uint64
	Level() uint64
	Scale() uint64
}

type ckksElement struct {
	value          []*ring.Poly
	scale          uint64
	currentModulus *ring.Int
	isNTT          bool
}

func (ckkscontext *Context) newCkksElement(degree, level, scale uint64) *ckksElement {
	el := &ckksElement{}
	el.value = make([]*ring.Poly, degree+1)
	for i := uint64(0); i < degree+1; i++ {
		el.value[i] = ckkscontext.contextLevel[level].NewPoly()
	}

	el.scale = scale
	el.currentModulus = ring.Copy(ckkscontext.contextLevel[level].ModulusBigint)
	el.isNTT = true

	return el

}

func (el *ckksElement) Value() []*ring.Poly {
	return el.value
}

func (el *ckksElement) SetValue(value []*ring.Poly) {
	el.value = value
}

func (el *ckksElement) Degree() uint64 {
	return uint64(len(el.value) - 1)
}

func (el *ckksElement) Level() uint64 {
	return uint64(len(el.value[0].Coeffs) - 1)
}

func (el *ckksElement) Scale() uint64 {
	return el.scale
}

func (el *ckksElement) SetScale(scale uint64) {
	el.scale = scale
}

func (el *ckksElement) CurrentModulus() *ring.Int {
	return el.currentModulus
}

func (el *ckksElement) SetCurrentModulus(modulus *ring.Int) {
	el.currentModulus = ring.Copy(modulus)
}

func (el *ckksElement) Resize(ckkscontext *Context, degree uint64) {
	if el.Degree() > degree {
		el.value = el.value[:degree+1]
	} else if el.Degree() < degree {
		for el.Degree() < degree {
			el.value = append(el.value, []*ring.Poly{ckkscontext.contextLevel[el.Level()].NewPoly()}...)
		}
	}
}

func (el *ckksElement) IsNTT() bool {
	return el.isNTT
}

func (el *ckksElement) SetIsNTT(value bool) {
	el.isNTT = value
}

// NTT puts the target element in the NTT domain and sets its isNTT flag to true. If it is already in the NTT domain, does nothing.
func (el *ckksElement) NTT(ckkscontext *Context, c *ckksElement) error {
	if el.Degree() != c.Degree() {
		return errors.New("error : receiver element invalide degree (does not match)")
	}
	if el.IsNTT() != true {
		for i := range el.value {
			ckkscontext.contextLevel[el.Level()].NTT(el.Value()[i], c.Value()[i])
		}
		c.SetIsNTT(true)
	}
	return nil
}

// InvNTT puts the target element outside of the NTT domain, and sets its isNTT flag to false. If it is not in the NTT domain, does nothing.
func (el *ckksElement) InvNTT(ckkscontext *Context, c *ckksElement) error {
	if el.Degree() != c.Degree() {
		return errors.New("error : receiver element invalide degree (does not match)")
	}
	if el.IsNTT() != false {
		for i := range el.value {
			ckkscontext.contextLevel[el.Level()].InvNTT(el.Value()[i], c.Value()[i])
		}
		c.SetIsNTT(false)
	}
	return nil
}

// CopyNew creates a new element which is a copy of the target element.
func (el *ckksElement) CopyNew() *ckksElement {

	ctxCopy := new(ckksElement)

	ctxCopy.value = make([]*ring.Poly, el.Degree()+1)
	for i := range el.value {
		ctxCopy.value[i] = el.value[i].CopyNew()
	}

	ctxCopy.CopyParams(el)

	return ctxCopy
}

// Copy copies the input element and its parameters on the target element.
func (el *ckksElement) Copy(ctxCopy *ckksElement) (err error) {

	if el != ctxCopy {
		for i := range ctxCopy.Value() {
			el.value[i].Copy(ctxCopy.Value()[i])
		}

		el.CopyParams(ctxCopy)
	}
	return nil
}

// CopyParams copies the input element parameters on the target element
func (el *ckksElement) CopyParams(ckkselement *ckksElement) {
	el.SetCurrentModulus(ckkselement.CurrentModulus())
	el.SetScale(ckkselement.Scale())
	el.SetIsNTT(ckkselement.IsNTT())
}

func (el *ckksElement) Element() *ckksElement {
	return el
}

func (el *ckksElement) Ciphertext() *Ciphertext {
	if len(el.value) == 1 {
		panic("not a ciphertext element")
	}
	return &Ciphertext{el}
}

func (el *ckksElement) Plaintext() *Plaintext {
	if len(el.value) != 1 {
		panic("not a plaintext element")
	}
	return &Plaintext{el, el.value[0]}
}
