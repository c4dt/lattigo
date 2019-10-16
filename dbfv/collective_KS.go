package dbfv

import (
	"github.com/ldsec/lattigo/ring"
)

// CKS is a structure storing the parameters for the collective key-switching protocol.
type CKS struct {
	context *ring.Context

	sigmaSmudging   float64
	gaussianSampler *ring.KYSampler

	deltaSk *ring.Poly
}

// NewCKS creates a new CKS that will be used to operate a collective key-switching on a ciphertext encrypted under a collective public-key, whose
// secret-shares are distributed among j parties, re-encrypting the ciphertext under another public-key, whose secret-shares are also known to the
// parties.
func NewCKS(skInput, skOutput *ring.Poly, context *ring.Context, sigmaSmudging float64) *CKS {

	cks := new(CKS)
	cks.context = context

	cks.sigmaSmudging = sigmaSmudging
	cks.gaussianSampler = context.NewKYSampler(sigmaSmudging, int(6*sigmaSmudging))

	cks.deltaSk = cks.context.NewPoly()
	context.Sub(skInput, skOutput, cks.deltaSk)
	return cks
}

// KeySwitch is the first and unique round of the CKS protocol. Each party holding a ciphertext ctx encrypted under a collective public-key must
// compute the following :
//
// [(skInput_i - skOutput_i) * ctx[0] + e_i]
//
// Each party then broadcasts the result of this computation to the other j-1 parties.
func (cks *CKS) KeySwitch(c1 *ring.Poly) *ring.Poly {

	h := c1.CopyNew()

	cks.context.NTT(h, h)
	cks.context.MulCoeffsMontgomery(h, cks.deltaSk, h)
	cks.context.InvNTT(h, h)
	cks.context.Add(h, cks.gaussianSampler.SampleNew(), h)

	return h
}

// Aggregate is the second part of the unique round of the CKS protocol. Upon receiving the j-1 elements each party computes :
//
// [ctx[0] + sum((skInput_i - skOutput_i) * ctx[0] + e_i), ctx[1]]
func (cks *CKS) Aggregate(c0 *ring.Poly, h []*ring.Poly) {

	for i := range h {
		cks.context.AddNoMod(c0, h[i], c0)

		if i&7 == 1 {
			cks.context.Reduce(c0, c0)
		}
	}

	if len(h)&7 != 7 {
		cks.context.Reduce(c0, c0)
	}
}
