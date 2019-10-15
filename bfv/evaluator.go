package bfv

import (
	"errors"
	"github.com/ldsec/lattigo/ring"
)

// Evaluator is a struct holding the necessary elements to operates the homomorphic operations between ciphertext and/or plaintexts.
// It also holds a small memory pool used to store intermediate computations.
type Evaluator struct {
	bfvcontext    *Context
	basisextender *ring.BasisExtender
	complexscaler *ring.ComplexScaler
	polypool      [4]*ring.Poly
	ctxpool       [3]*Ciphertext
}

// NewEvaluator creates a new Evaluator, that can be used to do homomorphic
// operations on the ciphertexts and/or plaintexts. It stores a small pool of polynomials
// and ciphertexts that will be used for intermediate values.
func (bfvcontext *Context) NewEvaluator() (evaluator *Evaluator) {

	evaluator = new(Evaluator)
	evaluator.bfvcontext = bfvcontext

	evaluator.basisextender = ring.NewBasisExtender(bfvcontext.contextQ, bfvcontext.contextP)
	evaluator.complexscaler = ring.NewComplexScaler(bfvcontext.t, bfvcontext.contextQ, bfvcontext.contextP)

	for i := 0; i < 4; i++ {
		evaluator.polypool[i] = bfvcontext.contextQP.NewPoly()
	}

	evaluator.ctxpool[0] = bfvcontext.NewCiphertextBig(5)
	evaluator.ctxpool[1] = bfvcontext.NewCiphertextBig(5)
	evaluator.ctxpool[2] = bfvcontext.NewCiphertextBig(5)

	return evaluator
}

func (evaluator *Evaluator) getElemAndCheckBinary(op0, op1, opOut Operand, opOutMinDegree uint64) (el0, el1, elOut *bfvElement, err error) {
	if op0 == nil || op1 == nil || opOut == nil {
		return nil, nil, nil, errors.New("operands cannot be nil")
	}

	if op0.Degree()+op1.Degree() == 0 {
		return nil, nil, nil, errors.New("operands cannot be both plaintext")
	}

	if opOut.Degree() < opOutMinDegree {
		return nil, nil, nil, errors.New("receiver operand degree is too small")
	}

	el0, el1, elOut = op0.Element(), op1.Element(), opOut.Element()
	return // TODO: more checks on elements
}

func (evaluator *Evaluator) getElemAndCheckUnary(op0, opOut Operand, opOutMinDegree uint64) (el0, elOut *bfvElement, err error) {
	if op0 == nil || opOut == nil {
		return nil, nil, errors.New("operand cannot be nil")
	}

	if op0.Degree() == 0 {
		return nil, nil, errors.New("operand cannot be plaintext")
	}

	if opOut.Degree() < opOutMinDegree {
		return nil, nil, errors.New("receiver operand degree is too small")
	}
	el0, elOut = op0.Element(), opOut.Element()
	return // TODO: more checks on elements
}

// evaluateInPlaceBinary applies the provided function in place on el0 and el1 and returns the result in elOut.
func evaluateInPlaceBinary(el0, el1, elOut *bfvElement, evaluate func(*ring.Poly, *ring.Poly, *ring.Poly)) {

	maxDegree := max(el0.Degree(), el1.Degree())
	minDegree := min(el0.Degree(), el1.Degree())

	for i := uint64(0); i < minDegree+1; i++ {
		evaluate(el0.value[i], el1.value[i], elOut.value[i])
	}

	// If the inputs degree differ, copies the remaining degree on the receiver
	var largest *bfvElement
	if el0.Degree() > el1.Degree() {
		largest = el0
	} else if el1.Degree() > el0.Degree() {
		largest = el1
	}
	if largest != nil && largest != elOut { // checks to avoid unnecessary work.
		for i := minDegree + 1; i < maxDegree+1; i++ {
			elOut.value[i].Copy(largest.value[i])
		}
	}
}

// evaluateInPlaceUnary applies the provided function in place on el0 and returns the result in elOut.
func evaluateInPlaceUnary(el0, elOut *bfvElement, evaluate func(*ring.Poly, *ring.Poly)) {
	for i := range el0.value {
		evaluate(el0.value[i], elOut.value[i])
	}
}

// Add adds op0 to op1 and returns the result on ctOut.
func (evaluator *Evaluator) Add(op0, op1 Operand, ctOut *Ciphertext) (err error) {
	el0, el1, elOut, err := evaluator.getElemAndCheckBinary(op0, op1, ctOut, max(op0.Degree(), op1.Degree()))
	if err != nil {
		return err
	}
	evaluateInPlaceBinary(el0, el1, elOut, evaluator.bfvcontext.contextQ.Add)
	return
}

// AddNew adds op0 to op1 and creates a new element ctOut to store the result.
func (evaluator *Evaluator) AddNew(op0, op1 Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(max(op0.Degree(), op1.Degree()))
	return ctOut, evaluator.Add(op0, op1, ctOut)
}

// AddNoMod adds op0 to op1 without modular reduction, and returns the result on cOut.
func (evaluator *Evaluator) AddNoMod(op0, op1 Operand, ctOut *Ciphertext) (err error) {
	el0, el1, elOut, err := evaluator.getElemAndCheckBinary(op0, op1, ctOut, max(op0.Degree(), op1.Degree()))
	if err != nil {
		return err
	}
	evaluateInPlaceBinary(el0, el1, elOut, evaluator.bfvcontext.contextQ.AddNoMod)
	return nil
}

// AddNoModNew adds op0 to op1 without modular reduction and creates a new element ctOut to store the result.
func (evaluator *Evaluator) AddNoModNew(op0, op1 Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(max(op0.Degree(), op1.Degree()))
	return ctOut, evaluator.AddNoMod(op0, op1, ctOut)
}

// Sub subtracts op1 to op0 and returns the result on cOut.
func (evaluator *Evaluator) Sub(op0, op1 Operand, ctOut *Ciphertext) (err error) {
	el0, el1, elOut, err := evaluator.getElemAndCheckBinary(op0, op1, ctOut, max(op0.Degree(), op1.Degree()))
	if err != nil {
		return err
	}
	evaluateInPlaceBinary(el0, el1, elOut, evaluator.bfvcontext.contextQ.Sub)
	return nil
}

// SubNew subtracts op0 to op1 and creates a new element ctOut to store the result.
func (evaluator *Evaluator) SubNew(op0, op1 Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(max(op0.Degree(), op1.Degree()))
	return ctOut, evaluator.Sub(op0, op1, ctOut)
}

// SubNoMod subtracts op0 to op1 without modular reduction and returns the result on ctOut.
func (evaluator *Evaluator) SubNoMod(op0, op1 Operand, ctOut *Ciphertext) (err error) {
	el0, el1, elOut, err := evaluator.getElemAndCheckBinary(op0, op1, ctOut, max(op0.Degree(), op1.Degree()))
	if err != nil {
		return err
	}
	evaluateInPlaceBinary(el0, el1, elOut, evaluator.bfvcontext.contextQ.SubNoMod)
	return nil
}

// SubNoModNew subtracts op0 to op1 without modular reduction and creates a new element ctOut to store the result.
func (evaluator *Evaluator) SubNoModNew(op0, op1 Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(max(op0.Degree(), op1.Degree()))
	return ctOut, evaluator.SubNoMod(op0, op1, ctOut)
}

// Neg negates op and returns the result on ctOut.
func (evaluator *Evaluator) Neg(op Operand, ctOut *Ciphertext) error {
	el0, elOut, err := evaluator.getElemAndCheckUnary(op, ctOut, op.Degree())
	if err != nil {
		return err
	}
	evaluateInPlaceUnary(el0, elOut, evaluator.bfvcontext.contextQ.Neg)
	return nil
}

// NegNew negates op and creates a new element to store the result.
func (evaluator *Evaluator) NegNew(op Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(op.Degree())
	return ctOut, evaluator.Neg(op, ctOut)
}

// Reduce applies a modular reduction on op and returns the result on ctOut.
func (evaluator *Evaluator) Reduce(op Operand, ctOut *Ciphertext) error {
	el0, elOut, err := evaluator.getElemAndCheckUnary(op, ctOut, op.Degree())
	if err != nil {
		return err
	}
	evaluateInPlaceUnary(el0, elOut, evaluator.bfvcontext.contextQ.Reduce)
	return nil
}

// ReduceNew applies a modular reduction on op and creates a new element ctOut to store the result.
func (evaluator *Evaluator) ReduceNew(op Operand) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(op.Degree())
	return ctOut, evaluator.Reduce(op, ctOut)
}

// MulScalar multiplies op by an uint64 scalar and returns the result on ctOut.
func (evaluator *Evaluator) MulScalar(op Operand, scalar uint64, ctOut *Ciphertext) error {

	el0, elOut, err := evaluator.getElemAndCheckUnary(op, ctOut, op.Degree())
	if err != nil {
		return err
	}
	fun := func(el, elOut *ring.Poly) { evaluator.bfvcontext.contextQ.MulScalar(el, scalar, elOut) }
	evaluateInPlaceUnary(el0, elOut, fun)
	return nil
}

// MulScalarNew multiplies op by an uint64 scalar and creates a new element ctOut to store the result.
func (evaluator *Evaluator) MulScalarNew(op Operand, scalar uint64) (ctOut *Ciphertext, err error) {
	ctOut = evaluator.bfvcontext.NewCiphertext(op.Degree())
	return ctOut, evaluator.MulScalar(op, scalar, ctOut)
}

// tensorAndRescales computes (ct0 x ct1) * (t/Q) and stores the result on ctOut.
func (evaluator *Evaluator) tensorAndRescale(ct0, ct1, ctOut *bfvElement) {

	// Prepares the ciphertexts for the Tensoring by extending their
	// basis from Q to QP and transforming them in NTT form
	c0 := evaluator.ctxpool[0]
	c1 := evaluator.ctxpool[1]
	tmpCtOut := evaluator.ctxpool[2]

	if ct0 == ct1 {

		for i := range ct0.value {
			evaluator.basisextender.ExtendBasis(ct0.value[i], c0.value[i])
			evaluator.bfvcontext.contextQP.NTT(c0.value[i], c0.value[i])
		}

	} else {

		for i := range ct0.value {
			evaluator.basisextender.ExtendBasis(ct0.value[i], c0.value[i])
			evaluator.bfvcontext.contextQP.NTT(c0.value[i], c0.value[i])
		}

		for i := range ct1.value {
			evaluator.basisextender.ExtendBasis(ct1.value[i], c1.value[i])
			evaluator.bfvcontext.contextQP.NTT(c1.value[i], c1.value[i])
		}
	}

	// Tensoring : multiplies each elements of the ciphertexts together
	// and adds them to their correspongint position in the new ciphertext
	// based on their respective degree

	// Case where both BfvElements are of degree 1
	if ct0.Degree() == 1 && ct1.Degree() == 1 {

		c00 := evaluator.polypool[0]
		c01 := evaluator.polypool[1]

		d0 := tmpCtOut.value[0]
		d1 := tmpCtOut.value[1]
		d2 := tmpCtOut.value[2]

		evaluator.bfvcontext.contextQP.MForm(c0.value[0], c00)
		evaluator.bfvcontext.contextQP.MForm(c0.value[1], c01)

		// Squaring case
		if ct0 == ct1 {

			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c00, c0.value[0], d0) // c0 = c0[0]*c0[0]
			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c00, c0.value[1], d1) // c1 = 2*c0[0]*0[1]
			evaluator.bfvcontext.contextQP.Add(d1, d1, d1)
			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c01, c0.value[1], d2) // c2 = c0[1]*c0[1]

			// Normal case
		} else {

			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c00, c1.value[0], d0) // c0 = c0[0]*c0[0]
			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c00, c1.value[1], d1)
			evaluator.bfvcontext.contextQP.MulCoeffsMontgomeryAndAddNoMod(c01, c1.value[0], d1) // c1 = c0[0]*c1[1] + c0[1]*c1[0]
			evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c01, c1.value[1], d2)            // c2 = c0[1]*c1[1]
		}

		// Case where both BfvElements are not of degree 1
	} else {

		for i := 0; i < len(ct0.Value())+len(ct1.Value()); i++ {
			tmpCtOut.value[i].Zero()
		}

		// Squaring case
		if ct0 == ct1 {

			c00 := evaluator.ctxpool[1]

			for i := range ct0.value {
				evaluator.bfvcontext.contextQP.MForm(c0.value[i], c00.value[i])
			}

			for i := uint64(0); i < ct0.Degree()+1; i++ {
				for j := i + 1; j < ct0.Degree()+1; j++ {
					evaluator.bfvcontext.contextQP.MulCoeffsMontgomery(c00.value[i], c0.value[j], tmpCtOut.value[i+j])
					evaluator.bfvcontext.contextQP.Add(tmpCtOut.value[i+j], tmpCtOut.value[i+j], tmpCtOut.value[i+j])
				}
			}

			for i := uint64(0); i < ct0.Degree()+1; i++ {
				evaluator.bfvcontext.contextQP.MulCoeffsMontgomeryAndAdd(c00.value[i], c0.value[i], tmpCtOut.value[i<<1])
			}

			// Normal case
		} else {
			for i := range ct0.value {
				evaluator.bfvcontext.contextQP.MForm(c0.value[i], c0.value[i])
				for j := range ct1.value {
					evaluator.bfvcontext.contextQP.MulCoeffsMontgomeryAndAdd(c0.value[i], c1.value[j], tmpCtOut.value[i+j])
				}
			}
		}
	}

	// Applies the inverse NTT to the ciphertext, scales the down ciphertext
	// by t/q and reduces its basis from QP to Q
	for i := range ctOut.value {
		evaluator.bfvcontext.contextQP.InvNTT(tmpCtOut.value[i], tmpCtOut.value[i])
		evaluator.complexscaler.Scale(tmpCtOut.value[i], ctOut.value[i])
	}
}

// Mul multiplies op0 by op1 and returns the result on ctOut.
func (evaluator *Evaluator) Mul(op0 *Ciphertext, op1 Operand, ctOut *Ciphertext) (err error) {

	el0, el1, elOut, err := evaluator.getElemAndCheckBinary(op0, op1, ctOut, op0.Degree()+op1.Degree())
	if err != nil {
		return err
	}
	evaluator.tensorAndRescale(el0, el1, elOut)
	return nil
}

// MulNew multiplies op0 by op1 and creates a new element ctOut to store the result.
func (evaluator *Evaluator) MulNew(op0 *Ciphertext, op1 Operand) (ctOut *Ciphertext, err error) {

	ctOut = evaluator.bfvcontext.NewCiphertext(op0.Degree() + op1.Degree())
	return ctOut, evaluator.Mul(op0, op1, ctOut)
}

// relinearize is a method common to Relinearize and RelinearizeNew. It switches ct0 out in the NTT domain, applies the keyswitch, and returns the result out of the NTT domain.
func (evaluator *Evaluator) relinearize(ct0 *Ciphertext, evakey *EvaluationKey, ctOut *Ciphertext) {

	evaluator.bfvcontext.contextQ.NTT(ct0.value[0], ctOut.value[0])
	evaluator.bfvcontext.contextQ.NTT(ct0.value[1], ctOut.value[1])

	for deg := uint64(ct0.Degree()); deg > 1; deg-- {
		evaluator.switchKeys(ct0.value[deg], evakey.evakey[deg-2], ctOut)
	}

	ctOut.SetValue(ctOut.value[:2])

	evaluator.bfvcontext.contextQ.InvNTT(ctOut.value[0], ctOut.value[0])
	evaluator.bfvcontext.contextQ.InvNTT(ctOut.value[1], ctOut.value[1])
}

// Relinearize relinearizes the ciphertext ct0 of degree > 1 until it is of degree 1 and returns the result on cOut.
//
// Requires a correct evaluation key as additional input :
//
// - it must match the secret-key that was used to create the public key under which the current ct0 is encrypted.
//
// - it must be of degree high enough to relinearize the input ciphertext to degree 1 (ex. a ciphertext
// of degree 3 will require that the evaluation key stores the keys for both degree 3 and 2 ciphertexts).
func (evaluator *Evaluator) Relinearize(ct0 *Ciphertext, evakey *EvaluationKey, ctOut *Ciphertext) error {

	if int(ct0.Degree()-1) > len(evakey.evakey) {
		return errors.New("cannot relinearize -> input ciphertext degree too large to allow relinearization")
	}

	if ct0.Degree() < 2 {
		if ct0 != ctOut {
			ctOut.Copy(ct0.Element())
		}
		return nil
	}

	evaluator.relinearize(ct0, evakey, ctOut)
	return nil
}

// RelinearizeNew relinearizes the ciphertext ct0 of degree > 1 until it is of degree 1 and creates a new ciphertext to store the result.
//
// Requires a correct evaluation key as additional input :
//
// - it must match the secret-key that was used to create the public key under which the current ct0 is encrypted
//
// - it must be of degree high enough to relinearize the input ciphertext to degree 1 (ex. a ciphertext
// of degree 3 will require that the evaluation key stores the keys for both degree 3 and 2 ciphertexts).
func (evaluator *Evaluator) RelinearizeNew(ct0 *Ciphertext, evakey *EvaluationKey) (ctOut *Ciphertext, err error) {

	ctOut = evaluator.bfvcontext.NewCiphertext(1)

	return ctOut, evaluator.Relinearize(ct0, evakey, ctOut)
}

// SwitchKeys applies the key-switching procedure to the ciphertext ct0 and returns the result on ctOut. It requires as an additional input a valid switching-key :
// it must encrypt the target key under the public key under which ct0 is currently encrypted.
func (evaluator *Evaluator) SwitchKeys(ct0 *Ciphertext, switchkey *SwitchingKey, ctOut *Ciphertext) (err error) {

	if ct0.Degree() != 1 || ctOut.Degree() != 1 {
		return errors.New("cannot switchkeys -> input and output must be of degree 1 to allow key switching")
	}

	evaluator.switchKeysOutOfNTTDomain(ct0.value[0], ct0.value[1], switchkey, ctOut)

	return nil
}

// SwitchKeysNew applies the key-switching procedure to the ciphertext ct0 and creates a new ciphertext to store the result. It requires as an additional input a valid switching-key :
// it must encrypt the target key under the public key under which ct0 is currently encrypted.
func (evaluator *Evaluator) SwitchKeysNew(ct0 *Ciphertext, switchkey *SwitchingKey) (ctOut *Ciphertext, err error) {

	ctOut = evaluator.bfvcontext.NewCiphertext(1)
	return ctOut, evaluator.SwitchKeys(ct0, switchkey, ctOut)
}

// RotateColumns rotates the columns of ct0 by k positions to the left and returns the result on ctOut. As an additional input it requires a rotationkeys :
//
// - it must either store all the left and right power of 2 rotations or the specific rotation that is asked.
//
// If only the power of two rotations are stored, the numbers k and n/2-k will be decomposed in base 2 and the rotation with the least
// hamming weight will be chosen, then the specific rotation will be computed as a sum of powers of two rotations.
func (evaluator *Evaluator) RotateColumns(ct0 *Ciphertext, k uint64, evakey *RotationKeys, ctOut *Ciphertext) (err error) {

	k &= ((evaluator.bfvcontext.n >> 1) - 1)

	if k == 0 {
		ctOut.Copy(ct0.Element())
		return nil
	}

	if ct0.Degree() != 1 || ctOut.Degree() != 1 {
		return errors.New("cannot rotate -> input and or output must be of degree 1")
	}

	// Looks in the rotationkey if the corresponding rotation has been generated or if the input is a plaintext
	if evakey.evakeyRotColLeft[k] != nil {

		evaluator.permute(ct0, ct0.IsNTT(), evaluator.bfvcontext.galElRotColLeft[k], evakey.evakeyRotColLeft[k], ctOut)

		return nil
	}

	// If not looks if the left and right pow2 rotations have been generated
	hasPow2Rotations := true
	for i := uint64(1); i < evaluator.bfvcontext.n>>1; i <<= 1 {
		if evakey.evakeyRotColLeft[i] == nil || evakey.evakeyRotColRight[i] == nil {
			hasPow2Rotations = false
			break
		}
	}

	// If yes, computes the least amount of rotation between k to the left and n/2 -k to the right required to apply the demanded rotation
	if hasPow2Rotations {

		if hammingWeight64(k) <= hammingWeight64((evaluator.bfvcontext.n>>1)-k) {
			evaluator.rotateColumnsLPow2(ct0, k, evakey, ctOut)
		} else {
			evaluator.rotateColumnsRPow2(ct0, (evaluator.bfvcontext.n>>1)-k, evakey, ctOut)
		}

		return nil
	}

	// Else returns an error indicating that the keys have not been generated
	return errors.New("cannot rotate -> specific rotation and pow2 rotations have not been generated")
}

// rotateColumnsLPow2 applies the Galois Automorphism on the element, rotating the element by k positions to the left, returns the result on ctOut.
func (evaluator *Evaluator) rotateColumnsLPow2(ct0 *Ciphertext, k uint64, evakey *RotationKeys, ctOut *Ciphertext) {
	evaluator.rotateColumnsPow2(ct0, evaluator.bfvcontext.gen, k, evakey.evakeyRotColLeft, ctOut)
}

// rotateColumnsRPow2 applies the Galois Endomorphism on the element, rotating the element by k positions to the right, returns the result on ctOut.
func (evaluator *Evaluator) rotateColumnsRPow2(ct0 *Ciphertext, k uint64, evakey *RotationKeys, ctOut *Ciphertext) {
	evaluator.rotateColumnsPow2(ct0, evaluator.bfvcontext.genInv, k, evakey.evakeyRotColRight, ctOut)
}

// rotateColumnsPow2 rotates ct0 by k position (left or right depending on the input), decomposing k as a sum of power of 2 rotations, and returns the result on ctOut.
func (evaluator *Evaluator) rotateColumnsPow2(ct0 *Ciphertext, generator, k uint64, evakeyRotCol map[uint64]*SwitchingKey, ctOut *Ciphertext) {

	var mask, evakeyIndex uint64

	context := evaluator.bfvcontext.contextQ

	mask = (evaluator.bfvcontext.n << 1) - 1

	evakeyIndex = 1

	if ct0.IsNTT() {
		ctOut.Copy(ct0.Element())
	} else {
		context.NTT(ct0.value[0], ctOut.value[0])
		context.NTT(ct0.value[1], ctOut.value[1])
	}

	// Applies the galois automorphism and the switching-key process
	for k > 0 {

		if k&1 == 1 {

			evaluator.permute(ctOut, true, generator, evakeyRotCol[evakeyIndex], ctOut)
		}

		generator *= generator
		generator &= mask

		evakeyIndex <<= 1
		k >>= 1
	}

	if !ct0.IsNTT() {
		context.InvNTT(ctOut.value[0], ctOut.value[0])
		context.InvNTT(ctOut.value[1], ctOut.value[1])
	}
}

// RotateRows swaps the rows of ct0 and returns the result on ctOut.
func (evaluator *Evaluator) RotateRows(ct0 *Ciphertext, evakey *RotationKeys, ctOut *Ciphertext) error {

	if ct0.Degree() != 1 || ctOut.Degree() != 1 {
		return errors.New("cannot rotate -> input and or output degree must be of degree 1")
	}

	if evakey.evakeyRotRows == nil {
		return errors.New("cannot rotate -> rotation key not generated")
	}

	evaluator.permute(ct0, ct0.IsNTT(), evaluator.bfvcontext.galElRotRow, evakey.evakeyRotRows, ctOut)

	return nil
}

// InnerSum computs the inner sum of ct0 and returns the result on ctOut. It requires a rotation key storing all the left power of two rotations.
// The resulting vector will be of the form [sum, sum, .., sum, sum ].
func (evaluator *Evaluator) InnerSum(ct0 *Ciphertext, evakey *RotationKeys, ctOut *Ciphertext) error {

	if ct0.Degree() != 1 || ctOut.Degree() != 1 {
		return errors.New("cannot inner sum -> input and output must be of degree 1")
	}

	cTmp := evaluator.bfvcontext.NewCiphertext(1)

	ctOut.Copy(ct0.Element())

	for i := uint64(1); i < evaluator.bfvcontext.n>>1; i <<= 1 {
		if err := evaluator.RotateColumns(ctOut, i, evakey, cTmp); err != nil {
			return err
		}
		evaluator.Add(cTmp.bfvElement, ctOut, ctOut.Ciphertext())
	}

	if err := evaluator.RotateRows(ctOut, evakey, cTmp); err != nil {
		return err
	}
	evaluator.Add(ctOut, cTmp.bfvElement, ctOut)

	return nil

}

// permute operates a column rotation on ct0 and returns the result on ctOut
func (evaluator *Evaluator) permute(ct0 *Ciphertext, isNTT bool, generator uint64, evakey *SwitchingKey, ctOut *Ciphertext) {

	context := evaluator.bfvcontext.contextQ

	var el0, el1 *ring.Poly

	if ct0 != ctOut {
		el0, el1 = ctOut.value[0], ctOut.value[1]
	} else {
		el0, el1 = evaluator.polypool[0], evaluator.polypool[1]
	}

	if isNTT {
		ring.PermuteNTT(ct0.value[0], generator, el0)
		ring.PermuteNTT(ct0.value[1], generator, el1)
		evaluator.switchKeysInNTTDomain(el0, el1, evakey, ctOut)
	} else {
		context.Permute(ct0.value[0], generator, el0)
		context.Permute(ct0.value[1], generator, el1)
		evaluator.switchKeysOutOfNTTDomain(el0, el1, evakey, ctOut)
	}
}

// switchKeysInNTTDomain operates a keyswitching assuming el0 and el1 are in the NTT domain
func (evaluator *Evaluator) switchKeysInNTTDomain(el0, el1 *ring.Poly, switchkey *SwitchingKey, ctOut *Ciphertext) {

	context := evaluator.bfvcontext.contextQ

	context.Copy(el0, ctOut.value[0])
	context.Copy(el1, ctOut.value[1])

	context.InvNTT(el1, evaluator.polypool[1])
	evaluator.switchKeys(evaluator.polypool[1], switchkey, ctOut)

}

// switchKeysOutOfNTTDomain operates a keyswitching assuming el0, el1 are not in the NTT domain
func (evaluator *Evaluator) switchKeysOutOfNTTDomain(el0, el1 *ring.Poly, switchKey *SwitchingKey, ctOut *Ciphertext) {

	context := evaluator.bfvcontext.contextQ

	context.Copy(el1, evaluator.polypool[1])

	context.NTT(el0, ctOut.value[0])
	context.NTT(el1, ctOut.value[1])

	evaluator.switchKeys(evaluator.polypool[1], switchKey, ctOut)

	context.InvNTT(ctOut.value[0], ctOut.value[0])
	context.InvNTT(ctOut.value[1], ctOut.value[1])
}

// switchKeys compute ctOut = [ctOut[0] + c2*evakey[0], ctOut[1] + c2*evakey[1]], for c2 not in NTT and ctOut in NTT.
func (evaluator *Evaluator) switchKeys(c2 *ring.Poly, evakey *SwitchingKey, ctOut *Ciphertext) {

	var mask, reduce, bitLog uint64

	c2qiw := evaluator.polypool[3]

	mask = uint64((1 << evakey.bitDecomp) - 1)

	reduce = 0

	for i := range evaluator.bfvcontext.contextQ.Modulus {

		bitLog = uint64(len(evakey.evakey[i]))

		for j := uint64(0); j < bitLog; j++ {
			//c2qiw = (c2qiw >> (w*z)) & (w-1)
			for u := uint64(0); u < evaluator.bfvcontext.n; u++ {
				for v := range evaluator.bfvcontext.contextQ.Modulus {
					c2qiw.Coeffs[v][u] = (c2.Coeffs[i][u] >> (j * evakey.bitDecomp)) & mask
				}
			}

			evaluator.bfvcontext.contextQ.NTT(c2qiw, c2qiw)

			evaluator.bfvcontext.contextQ.MulCoeffsMontgomeryAndAddNoMod(evakey.evakey[i][j][0], c2qiw, ctOut.value[0])
			evaluator.bfvcontext.contextQ.MulCoeffsMontgomeryAndAddNoMod(evakey.evakey[i][j][1], c2qiw, ctOut.value[1])

			if reduce&7 == 7 {
				evaluator.bfvcontext.contextQ.Reduce(ctOut.value[0], ctOut.value[0])
				evaluator.bfvcontext.contextQ.Reduce(ctOut.value[1], ctOut.value[1])
			}

			reduce++
		}
	}

	if (reduce-1)&7 != 7 {
		evaluator.bfvcontext.contextQ.Reduce(ctOut.value[0], ctOut.value[0])
		evaluator.bfvcontext.contextQ.Reduce(ctOut.value[1], ctOut.value[1])
	}
}
