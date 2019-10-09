package ckks

import (
	"fmt"
	"github.com/ldsec/lattigo/ring"
	"math/cmplx"
	"math/rand"
	"testing"
	"time"
)

type CKKSTESTPARAMS struct {
	ckkscontext *Context
	encoder     *Encoder
	kgen        *KeyGenerator
	sk          *SecretKey
	pk          *PublicKey
	rlk         *EvaluationKey
	rotkey      *RotationKeys
	encryptorPk *Encryptor
	encryptorSk *Encryptor
	decryptor   *Decryptor
	evaluator   *Evaluator
}

func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randomComplex(min, max float64) complex128 {
	return complex(randomFloat(min, max), randomFloat(min, max))
}

func TestCKKS(t *testing.T) {

	rand.Seed(time.Now().UnixNano())

	var err error

	params := Parameters{9, []uint8{55, 49, 49, 49, 49, 49, 49, 49, 49, 49}, 49, 3.2}

	ckksTest := new(CKKSTESTPARAMS)

	if ckksTest.ckkscontext, err = NewContext(&params); err != nil {
		t.Error(err)
	}

	ckksTest.encoder = ckksTest.ckkscontext.NewEncoder()

	ckksTest.kgen = ckksTest.ckkscontext.NewKeyGenerator()

	ckksTest.sk, ckksTest.pk = ckksTest.kgen.NewKeyPair()

	ckksTest.rlk = ckksTest.kgen.NewRelinKey(ckksTest.sk, ckksTest.ckkscontext.Scale())

	ckksTest.rotkey = ckksTest.kgen.NewRotationKeysPow2(ckksTest.sk, 15, true)

	if ckksTest.encryptorPk, err = ckksTest.ckkscontext.NewEncryptorFromPk(ckksTest.pk); err != nil {
		t.Error(err)
	}

	if ckksTest.encryptorSk, err = ckksTest.ckkscontext.NewEncryptorFromSk(ckksTest.sk); err != nil {
		t.Error(err)
	}

	if ckksTest.decryptor, err = ckksTest.ckkscontext.NewDecryptor(ckksTest.sk); err != nil {
		t.Error(err)
	}

	ckksTest.evaluator = ckksTest.ckkscontext.NewEvaluator()

	testEncoder(ckksTest, t)

	testEncryptDecrypt(ckksTest, t)

	testAdd(ckksTest, t)
	testSub(ckksTest, t)

	testAddConst(ckksTest, t)
	testMulConst(ckksTest, t)
	testMultByConstAndAdd(ckksTest, t)

	testComplexOperations(ckksTest, t)

	testRescaling(ckksTest, t)
	testMul(ckksTest, t)

	if ckksTest.ckkscontext.Levels() > 8 {
		testFunctions(ckksTest, t)
	}

	testSwitchKeys(ckksTest, t)
	testConjugate(ckksTest, t)
	testRotColumns(ckksTest, t)

	testMarshalCiphertext(ckksTest, t)
	testMarshalSecretKey(ckksTest, t)
	testMarshalPublicKey(ckksTest, t)
	testMarshalEvaluationKey(ckksTest, t)
	testMarshalSwitchingKey(ckksTest, t)
	testMarshalRotationKey(ckksTest, t)

}

func newTestvectors(params *CKKSTESTPARAMS, a, b float64) (values []complex128, plaintext *Plaintext, ciphertext *Ciphertext, err error) {

	slots := 1 << (params.ckkscontext.logN - 1)

	values = make([]complex128, slots)

	for i := 0; i < slots; i++ {
		values[i] = randomComplex(a, b)
	}

	values[0] = complex(0.607538, 0.555668)

	plaintext = params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

	if err = params.encoder.EncodeComplex(plaintext, values); err != nil {
		return nil, nil, nil, err
	}

	ciphertext, err = params.encryptorPk.EncryptNew(plaintext)
	if err != nil {
		return nil, nil, nil, err
	}

	return values, plaintext, ciphertext, nil
}

func newTestvectorsReals(params *CKKSTESTPARAMS, a, b float64) (values []complex128, plaintext *Plaintext, ciphertext *Ciphertext, err error) {

	slots := 1 << (params.ckkscontext.logN - 1)

	values = make([]complex128, slots)

	for i := 0; i < slots; i++ {
		values[i] = complex(randomFloat(a, b), 0)
	}

	values[0] = complex(0.607538, 0)

	plaintext = params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

	if err = params.encoder.EncodeComplex(plaintext, values); err != nil {
		return nil, nil, nil, err
	}

	ciphertext, err = params.encryptorPk.EncryptNew(plaintext)
	if err != nil {
		return nil, nil, nil, err
	}

	return values, plaintext, ciphertext, nil
}

func verifyTestvectors(params *CKKSTESTPARAMS, valuesWant []complex128, element Operand, t *testing.T) (err error) {

	var plaintextTest *Plaintext
	var valuesTest []complex128

	if element.Degree() == 0 {

		plaintextTest = element.Element().Plaintext()

	} else {

		plaintextTest = params.decryptor.DecryptNew(element.Element().Ciphertext())
	}

	valuesTest = params.encoder.DecodeComplex(plaintextTest)

	var DeltaReal0, DeltaImag0, DeltaReal1, DeltaImag1 float64

	for i := range valuesWant {

		// Test for big values (> 1)
		DeltaReal0 = real(valuesWant[i]) / real(valuesTest[i])
		DeltaImag0 = imag(valuesWant[i]) / imag(valuesTest[i])

		// Test for small values (< 1)
		DeltaReal1 = real(valuesWant[i]) - real(valuesTest[i])
		DeltaImag1 = imag(valuesWant[i]) - imag(valuesTest[i])

		if DeltaReal1 < 0 {
			DeltaReal1 *= -1
		}
		if DeltaImag1 < 0 {
			DeltaImag1 *= -1
		}

		if (DeltaReal0 < 0.999 || DeltaReal0 > 1.001 || DeltaImag0 < 0.999 || DeltaImag0 > 1.001) && (DeltaReal1 > 0.001 || DeltaImag1 > 0.001) {
			t.Errorf("error : coeff %d, want %f have %f", i, valuesWant[i], valuesTest[i])
			break
		}
	}

	return nil
}

func testEncoder(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/EncodeDecodeFloat64", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {
		slots := 1 << (params.ckkscontext.logN - 1)

		valuesWant := make([]float64, slots)
		valuesWantCmplx := make([]complex128, slots)
		for i := 0; i < slots; i++ {
			valuesWant[i] = randomFloat(0.000001, 5)
			valuesWantCmplx[i] = complex(valuesWant[i], 0)
		}

		plaintext := params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

		if err := params.encoder.EncodeFloat(plaintext, valuesWant); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWantCmplx, plaintext, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/EncodeDecodeComplex128", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {
		slots := 1 << (params.ckkscontext.logN - 1)

		valuesWant := make([]complex128, slots)

		for i := 0; i < slots; i++ {
			valuesWant[i] = randomComplex(0, 5)
		}

		plaintext := params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

		if err := params.encoder.EncodeComplex(plaintext, valuesWant); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, plaintext, t); err != nil {
			t.Error(err)
		}
	})
}

func testEncryptDecrypt(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/EncryptFromPk", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {
		var err error

		slots := 1 << (params.ckkscontext.logN - 1)

		valuesWant := make([]complex128, slots)

		for i := 0; i < slots; i++ {
			valuesWant[i] = randomComplex(0, 5)
		}

		plaintext := params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

		if err = params.encoder.EncodeComplex(plaintext, valuesWant); err != nil {
			t.Error(err)
		}

		ciphertext, err := params.encryptorPk.EncryptNew(plaintext)
		if err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/EncryptFromSk", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {
		var err error

		slots := 1 << (params.ckkscontext.logN - 1)

		valuesWant := make([]complex128, slots)

		for i := 0; i < slots; i++ {
			valuesWant[i] = randomComplex(0, 5)
		}

		plaintext := params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

		if err = params.encoder.EncodeComplex(plaintext, valuesWant); err != nil {
			t.Error(err)
		}

		ciphertext, err := params.encryptorSk.EncryptNew(plaintext)
		if err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext, t); err != nil {
			t.Error(err)
		}
	})
}

func testAdd(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/AddCtCtInPlace", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] + values2[i]
		}

		if err := params.evaluator.Add(ciphertext1, ciphertext2, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/AddCtCt", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		receiver := params.ckkscontext.NewCiphertext(1, ciphertext1.Level(), ciphertext1.Scale())

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] + values2[i]
		}

		if err := params.evaluator.Add(ciphertext1, ciphertext2, receiver); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, receiver, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Add(Ct,Plain)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, plaintext2, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] + values2[i]
		}

		if err := params.evaluator.Add(ciphertext1, plaintext2, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Add(Plain,Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, plaintext1, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] + values2[i]
		}

		if err := params.evaluator.Add(plaintext1, ciphertext2, ciphertext2); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext2, t); err != nil {
			t.Error(err)
		}
	})
}

func testSub(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Sub(Ct,Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] - values2[i]
		}

		if err := params.evaluator.Sub(ciphertext1, ciphertext2, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Sub(Ct,Plain)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, plaintext2, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] - values2[i]
		}

		if err := params.evaluator.Sub(ciphertext1, plaintext2, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Sub(Plain,Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, plaintext1, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] - values2[i]
		}

		if err := params.evaluator.Sub(plaintext1, ciphertext2, ciphertext2); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext2, t); err != nil {
			t.Error(err)
		}
	})

}

func testAddConst(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/AddCmplx(Ct,complex128)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		constant := complex(3.1415, -1.4142)

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] + constant
		}

		if err := params.evaluator.AddConst(ciphertext1, constant, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})
}

func testMulConst(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MultCmplx(Ct,complex128)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -5, 5)
		if err != nil {
			t.Error(err)
		}

		constant := complex(1.4142, -3.1415)

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] * (1 / constant)
		}

		if err = params.evaluator.MultConst(ciphertext1, 1/constant, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MultByi", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values[i] * complex(0, 1)
		}

		if err = params.evaluator.MultByi(ciphertext1, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/DivByi", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values[i] / complex(0, 1)
		}

		if err = params.evaluator.DivByi(ciphertext1, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})
}

func testMultByConstAndAdd(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MultByCmplxAndAdd(Ct0, complex128, Ct1)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		constant := complex(3.1415, -1.4142)

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		for i := 0; i < len(valuesWant); i++ {
			values2[i] += (values1[i] * constant) + (values1[i] * constant)
		}

		if err = params.evaluator.MultByConstAndAdd(ciphertext1, constant, ciphertext2); err != nil {
			t.Error(err)
		}

		params.evaluator.Rescale(ciphertext1, ciphertext1)

		if err = params.evaluator.MultByConstAndAdd(ciphertext1, constant, ciphertext2); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, values2, ciphertext2, t); err != nil {
			t.Error(err)
		}
	})
}

func testComplexOperations(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/ExtractImag", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < len(values); i++ {
			values[i] = complex(imag(values[i]), 0)
		}

		if err = params.evaluator.ExtractImag(ciphertext, params.rotkey, ciphertext); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, values, ciphertext, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/SwapRealImag", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < len(values); i++ {
			values[i] = complex(imag(values[i]), real(values[i]))
		}

		if err = params.evaluator.SwapRealImag(ciphertext, params.rotkey, ciphertext); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, values, ciphertext, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/RemoveReal", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < len(values); i++ {
			values[i] = complex(0, imag(values[i]))
		}

		if err = params.evaluator.RemoveReal(ciphertext, params.rotkey, ciphertext); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, values, ciphertext, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/RemoveImag", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < len(values); i++ {
			values[i] = complex(real(values[i]), 0)
		}

		if err = params.evaluator.RemoveImag(ciphertext, params.rotkey, ciphertext); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, values, ciphertext, t); err != nil {
			t.Error(err)
		}
	})
}

func testRescaling(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Rescaling", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		coeffs := make([]*ring.Int, params.ckkscontext.n)
		for i := uint64(0); i < params.ckkscontext.n; i++ {
			coeffs[i] = ring.RandInt(params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].ModulusBigint)
			coeffs[i].Div(coeffs[i], ring.NewUint(10))
		}

		coeffsWant := make([]*ring.Int, params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].N)
		for i := range coeffs {
			coeffsWant[i] = ring.Copy(coeffs[i])
			coeffsWant[i].Div(coeffsWant[i], ring.NewUint(params.ckkscontext.moduli[len(params.ckkscontext.moduli)-1]))
		}

		polTest := params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].NewPoly()
		polWant := params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].NewPoly()

		params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].SetCoefficientsBigint(coeffs, polTest)
		params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].SetCoefficientsBigint(coeffsWant, polWant)

		params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].NTT(polTest, polTest)
		params.ckkscontext.contextLevel[params.ckkscontext.Levels()-1].NTT(polWant, polWant)

		rescale(params.evaluator, polTest, polTest)

		for i := uint64(0); i < params.ckkscontext.n; i++ {
			for j := 0; j < len(params.ckkscontext.moduli)-1; j++ {
				if polWant.Coeffs[j][i] != polTest.Coeffs[j][i] {
					t.Errorf("error : coeff %v Qi%v = %s, want %v have %v", i, j, coeffs[i].String(), polWant.Coeffs[j][i], polTest.Coeffs[j][i])
					break
				}
			}
		}
	})
}

func testMul(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Mul(Ct,Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] * values2[i]
		}

		if err = params.evaluator.MulRelin(ciphertext1, ciphertext2, nil, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Relinearize", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i] * values2[i]
		}

		if err = params.evaluator.MulRelin(ciphertext1, ciphertext2, nil, ciphertext1); err != nil {
			t.Error(err)
		}

		if err = params.evaluator.Relinearize(ciphertext1, params.rlk, ciphertext1); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MulRelin(Ct,Ct)->Rescale", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i]
		}

		for i := uint64(0); i < params.ckkscontext.Levels()-1; i++ {

			for i := 0; i < len(valuesWant); i++ {
				valuesWant[i] *= values2[i]
			}

			if err = params.evaluator.MulRelin(ciphertext1, ciphertext2, params.rlk, ciphertext1); err != nil {
				t.Error(err)
			}

			if err = params.evaluator.Rescale(ciphertext1, ciphertext1); err != nil {
				t.Error(err)
			}
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MulRelin(Ct,Plain)->Rescale", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, plaintext2, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i]
		}

		for i := uint64(0); i < params.ckkscontext.Levels()-1; i++ {

			for i := 0; i < len(valuesWant); i++ {
				valuesWant[i] *= values2[i]
			}

			if err = params.evaluator.MulRelin(ciphertext1, plaintext2, params.rlk, ciphertext1); err != nil {
				t.Error(err)
			}

			if err = params.evaluator.Rescale(ciphertext1, ciphertext1); err != nil {
				t.Error(err)
			}
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MulRelin(Plain,Ct)->Rescale", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, plaintext1, _, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		values2, _, ciphertext2, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values2[i]
		}

		for i := uint64(0); i < params.ckkscontext.Levels()-1; i++ {

			for i := 0; i < len(valuesWant); i++ {
				valuesWant[i] *= values1[i]
			}

			if err = params.evaluator.MulRelin(plaintext1, ciphertext2, params.rlk, ciphertext2); err != nil {
				t.Error(err)
			}

			if err = params.evaluator.Rescale(ciphertext2, ciphertext2); err != nil {
				t.Error(err)
			}
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext2, t); err != nil {
			t.Error(err)
		}
	})
}

func testFunctions(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Square", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i]
		}

		for i := uint64(0); i < 1; i++ {

			for j := 0; j < len(valuesWant); j++ {
				valuesWant[j] *= valuesWant[j]
			}

			if err = params.evaluator.MulRelin(ciphertext1, ciphertext1, params.rlk, ciphertext1); err != nil {
				t.Error(err)
			}

			params.evaluator.Rescale(ciphertext1, ciphertext1)

		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/PowerOf2", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		// up to a level equal to 2 modulus
		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = values1[i]
		}

		var n uint64

		n = 2

		for i := uint64(0); i < n; i++ {
			for j := 0; j < len(valuesWant); j++ {
				valuesWant[j] *= valuesWant[j]
			}
		}

		if err = params.evaluator.PowerOf2(ciphertext1, n, params.rlk, ciphertext1); err != nil {
			t.Error(err)
		}

		if ciphertext1.Scale() >= 100 {
			params.evaluator.Rescale(ciphertext1, ciphertext1)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Power", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values1, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)
		tmp := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = complex(1, 0)
			tmp[i] = values1[i]
		}

		var n uint64

		n = 7

		for j := 0; j < len(valuesWant); j++ {
			for i := n; i > 0; i >>= 1 {

				if i&1 == 1 {
					valuesWant[j] *= tmp[j]
				}

				tmp[j] *= tmp[j]
			}
		}

		if err = params.evaluator.Power(ciphertext1, n, params.rlk, ciphertext1); err != nil {
			t.Error(err)
		}

		if ciphertext1.Scale() >= 100 {
			params.evaluator.Rescale(ciphertext1, ciphertext1)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Inverse", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectorsReals(params, 0.1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = complex(1, 0) / values[i]
		}

		if ciphertext1, err = params.evaluator.InverseNew(ciphertext1, 7, params.rlk); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/sin(x) [-1-1i, 1+1i] deg16", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectors(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = cmplx.Sin(values[i])
		}

		cheby := Approximate(cmplx.Sin, complex(-1, -1), complex(1, 1), 16)

		if ciphertext1, err = params.evaluator.EvaluateCheby(ciphertext1, cheby, params.rlk); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/exp(2*pi*i*x) [-1, 1] deg60", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectorsReals(params, -1, 1)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = exp2pi(values[i])
		}

		cheby := Approximate(exp2pi, -1, 1, 60)

		if ciphertext1, err = params.evaluator.EvaluateCheby(ciphertext1, cheby, params.rlk); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/sin(2*pi*x)/(2*pi) [-15, 15] deg128", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		values, _, ciphertext1, err := newTestvectorsReals(params, -15, 15)
		if err != nil {
			t.Error(err)
		}

		valuesWant := make([]complex128, params.ckkscontext.n>>1)

		for i := 0; i < len(valuesWant); i++ {
			valuesWant[i] = sin2pi2pi(values[i])
		}

		cheby := Approximate(sin2pi2pi, -15, 15, 128)

		if ciphertext1, err = params.evaluator.EvaluateCheby(ciphertext1, cheby, params.rlk); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext1, t); err != nil {
			t.Error(err)
		}
	})
}

func testSwitchKeys(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/SwitchKeys", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		slots := 1 << (params.ckkscontext.logN - 1)

		valuesWant := make([]complex128, slots)

		for i := 0; i < slots; i++ {
			valuesWant[i] = randomComplex(0, 5)
		}

		plaintext := params.ckkscontext.NewPlaintext(params.ckkscontext.Levels()-1, params.ckkscontext.Scale())

		if err := params.encoder.EncodeComplex(plaintext, valuesWant); err != nil {
			t.Error(err)
		}

		ciphertext, err := params.encryptorPk.EncryptNew(plaintext)
		if err != nil {
			t.Error(err)
		}

		sk2 := params.kgen.NewSecretKey()

		switchingkeys := params.kgen.NewSwitchingKey(params.sk, sk2, 10)

		if err = params.evaluator.SwitchKeys(ciphertext, switchingkeys, ciphertext); err != nil {
			t.Error(err)
		}

		decryptorSk2, err := params.ckkscontext.NewDecryptor(sk2)
		if err != nil {
			t.Error(err)
		}

		plaintextTest := decryptorSk2.DecryptNew(ciphertext)

		valuesTest := params.encoder.DecodeComplex(plaintextTest)

		var DeltaReal0, DeltaImag0, DeltaReal1, DeltaImag1 float64

		for i := range valuesWant {

			// Test for big values (> 1)
			DeltaReal0 = real(valuesWant[i]) / real(valuesTest[i])
			DeltaImag0 = imag(valuesWant[i]) / imag(valuesTest[i])

			// Test for small values (< 1)
			DeltaReal1 = real(valuesWant[i]) - real(valuesTest[i])
			DeltaImag1 = imag(valuesWant[i]) - imag(valuesTest[i])

			if DeltaReal1 < 0 {
				DeltaReal1 *= -1
			}
			if DeltaImag1 < 0 {
				DeltaImag1 *= -1
			}

			if (DeltaReal0 < 0.999 || DeltaReal0 > 1.001 || DeltaImag0 < 0.999 || DeltaImag0 > 1.001) && (DeltaReal1 > 0.001 || DeltaImag1 > 0.001) {
				t.Errorf("error : coeff %d, want %f have %f", i, valuesWant[i], valuesTest[i])
				break
			}
		}
	})
}

func testConjugate(params *CKKSTESTPARAMS, t *testing.T) {

	values, _, ciphertext, err := newTestvectorsReals(params, -15, 15)
	if err != nil {
		t.Error(err)
	}

	valuesWant := make([]complex128, len(values))
	for i := range values {
		valuesWant[i] = complex(real(values[i]), -imag(values[i]))
	}

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/Conjugate(Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		if err := params.evaluator.Conjugate(ciphertext, params.rotkey, ciphertext); err != nil {
			t.Error(err)
		}

		if err := verifyTestvectors(params, valuesWant, ciphertext, t); err != nil {
			t.Error(err)
		}
	})
}

func testRotColumns(params *CKKSTESTPARAMS, t *testing.T) {

	mask := params.ckkscontext.slots - 1

	values, _, ciphertext, err := newTestvectorsReals(params, 0.1, 1)
	if err != nil {
		t.Error(err)
	}

	valuesWant := make([]complex128, params.ckkscontext.slots)

	ciphertextTest := params.ckkscontext.NewCiphertext(1, ciphertext.Level(), ciphertext.Scale())
	ciphertextTest.SetScale(ciphertext.Scale())

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/RotColumnsPow2(Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		for n := uint64(1); n < params.ckkscontext.slots; n <<= 1 {

			// Applies the column rotation to the values
			for i := uint64(0); i < params.ckkscontext.slots; i++ {
				valuesWant[i] = values[(i+n)&mask]
			}

			if err := params.evaluator.RotateColumns(ciphertext, n, params.rotkey, ciphertextTest); err != nil {
				t.Error(err)
			}

			if err := verifyTestvectors(params, valuesWant, ciphertextTest, t); err != nil {
				t.Error(err)
			}
		}
	})

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/RotColumnsRandom(Ct)", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		for n := uint64(1); n < params.ckkscontext.slots; n <<= 1 {

			rand := ring.RandUniform(params.ckkscontext.slots, mask)

			// Applies the column rotation to the values
			for i := uint64(0); i < params.ckkscontext.slots; i++ {
				valuesWant[i] = values[(i+rand)&mask]
			}

			if err := params.evaluator.RotateColumns(ciphertext, rand, params.rotkey, ciphertextTest); err != nil {
				t.Error(err)
			}

			if err := verifyTestvectors(params, valuesWant, ciphertextTest, t); err != nil {
				t.Error(err)
			}
		}
	})
}

func testMarshalCiphertext(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalCiphertext", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		_, _, ciphertext, err := newTestvectorsReals(params, 0.1, 1)
		if err != nil {
			t.Error(err)
		}

		b, err := ciphertext.MarshalBinary()
		if err != nil {
			t.Error(err)
		}

		newCT := params.ckkscontext.NewCiphertext(ciphertext.Degree(), ciphertext.Level(), 0)

		err = newCT.UnMarshalBinary(b)
		if err != nil {
			t.Error(err)
		}

		if !params.ckkscontext.keyscontext.Equal(ciphertext.Value()[0], newCT.Value()[0]) {
			t.Errorf("marshal binary ciphertext[0]")
		}

		if !params.ckkscontext.keyscontext.Equal(ciphertext.Value()[1], newCT.Value()[1]) {
			t.Errorf("marshal binary ciphertext[1]")
		}
	})

}

func testMarshalSecretKey(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalSK", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		b, _ := params.sk.MarshalBinary()

		newSK := params.kgen.NewSecretKeyEmpty()
		if err := newSK.UnMarshalBinary(b); err != nil {
			t.Error(err)
		}

		if !params.ckkscontext.keyscontext.Equal(params.sk.sk, newSK.sk) {
			t.Errorf("marshal binary sk")
		}
	})
}

func testMarshalPublicKey(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalPK", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		b, _ := params.pk.MarshalBinary()

		newPK := params.kgen.NewPublicKeyEmpty()
		if err := newPK.UnMarshalBinary(b); err != nil {
			t.Error(err)
		}

		if !params.ckkscontext.keyscontext.Equal(params.pk.pk[0], newPK.pk[0]) {
			t.Errorf("marshal binary pk[0]")
		}

		if !params.ckkscontext.keyscontext.Equal(params.pk.pk[1], newPK.pk[1]) {
			t.Errorf("marshal binary pk[1]")
		}
	})
}

func testMarshalEvaluationKey(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalRlk", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		b, _ := params.rlk.MarshalBinary()

		newRlk := params.kgen.NewRelinKeyEmpty(params.rlk.evakey.bitDecomp)
		if err := newRlk.UnMarshalBinary(b); err != nil {
			t.Error(err)
		}

		for x := range newRlk.evakey.evakey {
			for j := range newRlk.evakey.evakey[x] {
				if !params.ckkscontext.keyscontext.Equal(newRlk.evakey.evakey[x][j][0], params.rlk.evakey.evakey[x][j][0]) {
					t.Errorf("marshal binary rlk[0]")
					break
				}

				if !params.ckkscontext.keyscontext.Equal(newRlk.evakey.evakey[x][j][1], params.rlk.evakey.evakey[x][j][1]) {
					t.Errorf("marshal binary rlk[1]")
					break
				}
			}
		}
	})
}

func testMarshalSwitchingKey(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalSwitchingKey", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		bitDecomp := uint64(15)

		s1 := params.kgen.NewSecretKey()

		switchkey := params.kgen.NewSwitchingKey(params.sk, s1, bitDecomp)

		b, _ := switchkey.MarshalBinary()

		newSwKey := params.kgen.NewSwitchingKeyEmpty(bitDecomp)
		if err := newSwKey.UnMarshalBinary(b); err != nil {
			t.Error(err)
		}

		for x := range newSwKey.evakey {
			for j := range newSwKey.evakey[x] {
				if !params.ckkscontext.keyscontext.Equal(newSwKey.evakey[x][j][0], switchkey.evakey[x][j][0]) {
					t.Errorf("marshal binary switchingkey[0]")
					break
				}

				if !params.ckkscontext.keyscontext.Equal(newSwKey.evakey[x][j][1], switchkey.evakey[x][j][1]) {
					t.Errorf("marshal binary switchingkey[1]")
					break
				}
			}
		}

	})
}

func testMarshalRotationKey(params *CKKSTESTPARAMS, t *testing.T) {

	t.Run(fmt.Sprintf("logN=%d/logQ=%d/levels=%d/MarshalRotKey", params.ckkscontext.logN,
		params.ckkscontext.logQ,
		params.ckkscontext.levels), func(t *testing.T) {

		b, _ := params.rotkey.MarshalBinary()

		newRotKey := params.kgen.NewRotationKeysEmpty()
		if err := newRotKey.UnMarshalBinary(b); err != nil {
			t.Error(err)
		}

		for i := range params.rotkey.evakeyRotColLeft {
			for x := range params.rotkey.evakeyRotColLeft[i].evakey {
				for j := range params.rotkey.evakeyRotColLeft[i].evakey[x] {
					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotColLeft[i].evakey[x][j][0], newRotKey.evakeyRotColLeft[i].evakey[x][j][0]) {
						t.Errorf("marshal binary rotkeyLeft[0]")
						break
					}

					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotColLeft[i].evakey[x][j][1], newRotKey.evakeyRotColLeft[i].evakey[x][j][1]) {
						t.Errorf("marshal binary rotkeyLeft[1]")
						break
					}
				}
			}
		}

		for i := range params.rotkey.evakeyRotColRight {
			for x := range params.rotkey.evakeyRotColRight[i].evakey {
				for j := range params.rotkey.evakeyRotColRight[i].evakey[x] {
					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotColRight[i].evakey[x][j][0], newRotKey.evakeyRotColRight[i].evakey[x][j][0]) {
						t.Errorf("marshal binary rotkeyRight[0]")
						break
					}

					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotColRight[i].evakey[x][j][1], newRotKey.evakeyRotColRight[i].evakey[x][j][1]) {
						t.Errorf("marshal binary rotkeyRight[1]")
						break
					}
				}
			}
		}

		if params.rotkey.evakeyRotRows != nil {
			for x := range params.rotkey.evakeyRotRows.evakey {
				for j := range params.rotkey.evakeyRotRows.evakey[x] {
					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotRows.evakey[x][j][0], newRotKey.evakeyRotRows.evakey[x][j][0]) {
						t.Errorf("marshal binary rotkeyConjugate[0]")
						break
					}

					if !params.ckkscontext.keyscontext.Equal(params.rotkey.evakeyRotRows.evakey[x][j][1], newRotKey.evakeyRotRows.evakey[x][j][1]) {
						t.Errorf("marshal binary rotkeyConjugate[1]")
						break
					}
				}
			}
		}
	})
}
