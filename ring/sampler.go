package ring

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
)

// KYSampler is the structure holding the parameters for the gaussian sampling.
type KYSampler struct {
	context *Context
	sigma   float64
	bound   int
	Matrix  [][]uint8
}

// NewKYSampler creates a new KYSampler with sigma and bound that will be used to sample polynomial within the provided discret gaussian distribution.
func (context *Context) NewKYSampler(sigma float64, bound int) *KYSampler {
	kysampler := new(KYSampler)
	kysampler.context = context
	kysampler.sigma = sigma
	kysampler.bound = bound
	kysampler.Matrix = computeMatrix(sigma, bound)
	return kysampler
}

//gaussian computes (1/variange*sqrt(pi)) * exp((x^2) / (2*variance^2)),  2.50662827463100050241576528481104525300698674060993831662992357 = sqrt(2*pi)
func gaussian(x, sigma float64) float64 {
	return (1 / (sigma * 2.5066282746310007)) * math.Exp(-((math.Pow(x, 2)) / (2 * sigma * sigma)))
}

// computeMatrix computes the binary expension with precision x in bits of the normal distribution
// with sigma and bound. Returns a matrix of the form M = [[0,1,0,0,...],[0,0,1,,0,...]],
// where each row is the binary expension of the normal distribution of index(row) with sigma and bound (center=0).
func computeMatrix(sigma float64, bound int) [][]uint8 {
	var g float64
	var x uint64

	precision := uint64(56)

	M := make([][]uint8, bound)

	breakCounter := 0

	for i := 0; i < bound; i++ {

		g = gaussian(float64(i), sigma)

		if i == 0 {
			g *= math.Exp2(float64(precision) - 1)
		} else {
			g *= math.Exp2(float64(precision))
		}

		x = uint64(g)

		if x == 0 {
			break
		}

		M[i] = make([]uint8, precision-1)

		for j := uint64(0); j < precision-1; j++ {
			M[i][j] = uint8((x >> (precision - j - 2)) & 1)
		}

		breakCounter++
	}

	M = M[:breakCounter]

	return M
}

func kysampling(M [][]uint8, randomBytes []byte, pointer uint8) (uint64, uint64, []byte, uint8) {

	var sign uint8

	d := 0
	col := 0
	colLen := len(M)

	for {

		// Uses one random byte per cycle and cycle through the randombytes
		for i := pointer; i < 8; i++ {

			d = (d << 1) + 1 - int((uint8(randomBytes[0])>>i)&1)

			// There is small probability that it will get out of the bound, then
			// rerun until it gets a proper output
			if d > colLen-1 {
				return kysampling(M, randomBytes, i)
			}

			for row := colLen - 1; row >= 0; row-- {

				d -= int(M[row][col])

				if d == -1 {

					// Sign
					if i == 7 {
						pointer = 0
						// If the last bit of the byte was read, samples a new byte for the sign
						randomBytes = randomBytes[1:]

						if len(randomBytes) == 0 {
							randomBytes = make([]byte, 8)
							if _, err := rand.Read(randomBytes); err != nil {
								panic("crypto rand error")
							}
						}

						sign = uint8(randomBytes[0]) & 1

					} else {
						pointer = i
						// Else the sign is the next bit of the byte
						sign = uint8(randomBytes[0]>>(i+1)) & 1
					}

					return uint64(row), uint64(sign), randomBytes, pointer + 1
				}
			}

			col++
		}

		// Resets the bit pointer and discards the used byte
		pointer = 0
		randomBytes = randomBytes[1:]

		// Sample 8 new bytes if the last byte was discarded
		if len(randomBytes) == 0 {
			randomBytes = make([]byte, 8)
			if _, err := rand.Read(randomBytes); err != nil {
				panic("crypto rand error")
			}
		}

	}
}

// SampleNew samples a new polynomial with gaussian distribution given the target kys parameters.
func (kys *KYSampler) SampleNew() *Poly {
	Pol := kys.context.NewPoly()
	kys.Sample(Pol)
	return Pol
}

// Sample samples on the target polynomial coefficients with gaussian distribution given the target kys parameters.
func (kys *KYSampler) Sample(Pol *Poly) {

	var coeff uint64
	var sign uint64

	randomBytes := make([]byte, 8)
	pointer := uint8(0)

	if _, err := rand.Read(randomBytes); err != nil {
		panic("crypto rand error")
	}

	for i := uint64(0); i < kys.context.N; i++ {

		coeff, sign, randomBytes, pointer = kysampling(kys.Matrix, randomBytes, pointer)

		for j, qi := range kys.context.Modulus {
			Pol.Coeffs[j][i] = (coeff & (sign * 0xFFFFFFFFFFFFFFFF)) | ((qi - coeff) & ((sign ^ 1) * 0xFFFFFFFFFFFFFFFF))

		}
	}
}

// SampleNTTNew samples a polynomial with gaussian distribution given the target kys context and apply the NTT.
func (kys *KYSampler) SampleNTTNew() *Poly {
	Pol := kys.SampleNew()
	kys.context.NTT(Pol, Pol)
	return Pol
}

// SampleNTT samples on the target polynomial coefficients with gaussian distribution given the target kys parameters,and applies the NTT.
func (kys *KYSampler) SampleNTT(Pol *Poly) {
	kys.Sample(Pol)
	kys.context.NTT(Pol, Pol)
}

// TernarySampler is the structure holding the parameters for sampling polynomials of the form [-1, 0, 1].
type TernarySampler struct {
	context          *Context
	Matrix           [][]uint64
	MatrixMontgomery [][]uint64

	KYMatrix [][]uint8
}

// NewTernarySampler creates a new TernarySampler from the target context.
func (context *Context) NewTernarySampler() *TernarySampler {

	sampler := new(TernarySampler)
	sampler.context = context

	sampler.Matrix = make([][]uint64, len(context.Modulus))
	sampler.MatrixMontgomery = make([][]uint64, len(context.Modulus))

	for i, Qi := range context.Modulus {

		sampler.Matrix[i] = make([]uint64, 3)
		sampler.Matrix[i][0] = 0
		sampler.Matrix[i][1] = 1
		sampler.Matrix[i][2] = Qi - 1

		sampler.MatrixMontgomery[i] = make([]uint64, 3)
		sampler.MatrixMontgomery[i][0] = 0
		sampler.MatrixMontgomery[i][1] = MForm(1, Qi, context.bredParams[i])
		sampler.MatrixMontgomery[i][2] = MForm(Qi-1, Qi, context.bredParams[i])
	}

	return sampler
}

func computeMatrixTernary(p float64) (M [][]uint8) {
	var g float64
	var x uint64

	precision := uint64(56)

	M = make([][]uint8, 2)

	g = p
	g *= math.Exp2(float64(precision))
	x = uint64(g)

	M[0] = make([]uint8, precision-1)
	for j := uint64(0); j < precision-1; j++ {
		M[0][j] = uint8((x >> (precision - j - 1)) & 1)
	}

	g = 1 - p
	g *= math.Exp2(float64(precision))
	x = uint64(g)

	M[1] = make([]uint8, precision-1)
	for j := uint64(0); j < precision-1; j++ {
		M[1][j] = uint8((x >> (precision - j - 1)) & 1)
	}

	return M
}

// SampleMontgomeryNew samples coefficients with ternary distribution in montgomery form on the target polynomial.
func (sampler *TernarySampler) sample(samplerMatrix [][]uint64, p float64, pol *Poly) (err error) {

	if p == 0 {
		return errors.New("cannot sample -> p = 0")
	}

	var coeff uint64
	var sign uint64
	var index uint64

	if p == 0.5 {

		randomBytesCoeffs := make([]byte, sampler.context.N>>3)
		randomBytesSign := make([]byte, sampler.context.N>>3)

		if _, err := rand.Read(randomBytesCoeffs); err != nil {
			panic("crypto rand error")
		}

		if _, err := rand.Read(randomBytesSign); err != nil {
			panic("crypto rand error")
		}

		for i := uint64(0); i < sampler.context.N; i++ {
			coeff = uint64(uint8(randomBytesCoeffs[i>>3])>>(i&7)) & 1
			sign = uint64(uint8(randomBytesSign[i>>3])>>(i&7)) & 1

			index = (coeff & (sign ^ 1)) | ((sign & coeff) << 1)

			for j := range sampler.context.Modulus {
				pol.Coeffs[j][i] = samplerMatrix[j][index] //(coeff & (sign^1)) | (qi - 1) * (sign & coeff)
			}
		}

	} else {

		matrix := computeMatrixTernary(p)

		randomBytes := make([]byte, 8)

		pointer := uint8(0)

		if _, err := rand.Read(randomBytes); err != nil {
			panic("crypto rand error")
		}

		for i := uint64(0); i < sampler.context.N; i++ {

			coeff, sign, randomBytes, pointer = kysampling(matrix, randomBytes, pointer)

			index = (coeff & (sign ^ 1)) | ((sign & coeff) << 1)

			for j := range sampler.context.Modulus {
				pol.Coeffs[j][i] = samplerMatrix[j][index] //(coeff & (sign^1)) | (qi - 1) * (sign & coeff)
			}
		}
	}

	return nil
}

// SampleUniform samples a polynomial with coefficients [-1, 0, 1] with a distribution of [1/3, 1/3, 1/3].
func (sampler *TernarySampler) SampleUniform(pol *Poly) {
	_ = sampler.sample(sampler.Matrix, 1.0/3.0, pol)
}

// Sample samples a polynomail with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2].
func (sampler *TernarySampler) Sample(p float64, pol *Poly) (err error) {
	if err = sampler.sample(sampler.Matrix, p, pol); err != nil {
		return err
	}
	return nil
}

// SampleMontgomery samples a polynomail with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in montgomery form.
func (sampler *TernarySampler) SampleMontgomery(p float64, pol *Poly) (err error) {
	if err = sampler.sample(sampler.MatrixMontgomery, p, pol); err != nil {
		return err
	}
	return nil
}

// SampleNew samples a new polynomial with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2]
func (sampler *TernarySampler) SampleNew(p float64) (pol *Poly, err error) {
	pol = sampler.context.NewPoly()
	if err = sampler.Sample(p, pol); err != nil {
		return nil, err
	}
	return pol, nil
}

// SampleMontgomeryNew samples a new polynomial with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in montgomery form.
func (sampler *TernarySampler) SampleMontgomeryNew(p float64) (pol *Poly, err error) {
	pol = sampler.context.NewPoly()
	if err = sampler.SampleMontgomery(p, pol); err != nil {
		return nil, err
	}
	return pol, nil
}

// SampleNTTNew samples a new polynomial with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in the NTT domain.
func (sampler *TernarySampler) SampleNTTNew(p float64) (pol *Poly, err error) {
	pol = sampler.context.NewPoly()
	if err = sampler.Sample(p, pol); err != nil {
		return nil, err
	}
	sampler.context.NTT(pol, pol)
	return pol, nil
}

// SampleNTT samples coefficients with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in the NTT domain.
func (sampler *TernarySampler) SampleNTT(p float64, pol *Poly) (err error) {
	if err = sampler.Sample(p, pol); err != nil {
		return err
	}
	sampler.context.NTT(pol, pol)

	return nil
}

// SampleMontgomeryNTTNew samples a new polynomial with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in the NTT domain and in montgomery form.
func (sampler *TernarySampler) SampleMontgomeryNTTNew(p float64) (pol *Poly, err error) {
	if pol, err = sampler.SampleMontgomeryNew(p); err != nil {
		return nil, err
	}
	sampler.context.NTT(pol, pol)
	return pol, nil
}

// SampleMontgomeryNTT samples coefficients with coefficients [-1, 0, 1] with a distribution of [(1-p)/2, p, (1-p)/2] in the NTT domain and in montgomery form.
func (sampler *TernarySampler) SampleMontgomeryNTT(p float64, pol *Poly) (err error) {
	if err = sampler.SampleMontgomery(p, pol); err != nil {
		return err
	}
	sampler.context.NTT(pol, pol)
	return nil
}

// RandUniform samples a uniform randomInt variable in the range [0, mask] until randomInt is in the range [0, v-1].
// mask needs to be of the form 2^n -1.
func RandUniform(v uint64, mask uint64) (randomInt uint64) {
	for {
		randomInt = randInt64(mask)
		if randomInt < v {
			return randomInt
		}
	}
}

// randInt3 samples a bit and a sign with rejection sampling (25% chance of failure), with probabilities :
// Pr[int = 0 : 1/3 ; int = 1 : 2/3]
// Pr[sign = 1 : 1/2; sign = 0 : 1/2]
func randInt3() (uint64, uint64) {
	var randomInt uint64

	for {
		randomInt = randInt8()
		if (randomInt & 3) < 3 {
			// (a|b) is 1/3 = 0 and 2/3 = 1 if (a<<1 | b) in [0,1,2]
			return ((randomInt >> 1) | randomInt) & 1, (randomInt >> 2) & 1
		}
	}
}

// randUint3 samples a uniform variable in the range [0, 2]
func randUint3() uint64 {
	var randomInt uint64
	for {
		randomInt = randInt8() & 3
		if randomInt < 3 {
			return randomInt
		}
	}
}

// randInt8 samples a uniform variable in the range [0, 255]
func randInt8() uint64 {

	// generate random 4 bytes
	randomBytes := make([]byte, 1)
	if _, err := rand.Read(randomBytes); err != nil {
		panic("crypto rand error")
	}
	// return required bits
	return uint64(randomBytes[0])
}

// randInt32 samples a uniform variable in the range [0, mask], where mask is of the form 2^n-1, with n in [0, 32].
func randInt32(mask uint64) uint64 {

	// generate random 4 bytes
	randomBytes := make([]byte, 4)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("crypto rand error")
	}

	// convert 4 bytes to a uint32
	randomUint32 := uint64(binary.BigEndian.Uint32(randomBytes))

	// return required bits
	return mask & randomUint32
}

// randInt64 samples a uniform variable in the range [0, mask], where mask is of the form 2^n-1, with n in [0, 64].
func randInt64(mask uint64) uint64 {

	// generate random 8 bytes
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("crypto rand error")
	}

	// convert 8 bytes to a uint64
	randomUint64 := binary.BigEndian.Uint64(randomBytes)

	// return required bits
	return mask & randomUint64
}
