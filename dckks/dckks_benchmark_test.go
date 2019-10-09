package dckks

import (
	"fmt"
	"github.com/ldsec/lattigo/ckks"
	"github.com/ldsec/lattigo/ring"
	"math"
	"testing"
)

type benchParams struct {
	parties       uint64
	params        *ckks.Parameters
	sigmaSmudging float64
	bdc           uint64
}

type benchContext struct {
	ckkscontext *ckks.Context
	sk0         *ckks.SecretKey
	sk1         *ckks.SecretKey
	pk0         *ckks.PublicKey
	pk1         *ckks.PublicKey
	cprng       *CRPGenerator
}

func BenchmarkDCKKSScheme(b *testing.B) {

	var err error

	params := []benchParams{

		{parties: 5, params: ckks.DefaultParams[14], sigmaSmudging: 6.4, bdc: 60},
	}

	for _, param := range params {

		benchcontext := new(benchContext)

		if benchcontext.ckkscontext, err = ckks.NewContext(param.params); err != nil {
			b.Error(err)
		}

		kgen := benchcontext.ckkscontext.NewKeyGenerator()

		benchcontext.sk0, benchcontext.pk0 = kgen.NewKeyPair()
		benchcontext.sk1, benchcontext.pk1 = kgen.NewKeyPair()

		benchcontext.cprng, err = NewCRPGenerator(nil, benchcontext.ckkscontext.ContextKeys())
		if err != nil {
			b.Error(err)
		}

		benchcontext.cprng.Seed([]byte{})

		benchEKG(param, benchcontext, b)
		benchEKGNaive(param, benchcontext, b)
		benchCKG(param, benchcontext, b)
		benchCKS(param, benchcontext, b)
		benchPCKS(param, benchcontext, b)

	}
}

func benchEKG(params benchParams, context *benchContext, b *testing.B) {
	// EKG
	bitLog := uint64(math.Ceil(float64(60) / float64(params.bdc)))

	EkgProtocol := NewEkgProtocol(context.ckkscontext.ContextKeys(), params.bdc)

	crp := make([][]*ring.Poly, context.ckkscontext.Levels())

	for i := uint64(0); i < context.ckkscontext.Levels(); i++ {
		crp[i] = make([]*ring.Poly, bitLog)
		for j := uint64(0); j < bitLog; j++ {
			crp[i][j] = context.cprng.Clock()
		}
	}

	samples := make([][][]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		samples[i] = make([][]*ring.Poly, context.ckkscontext.Levels())
		samples[i] = EkgProtocol.GenSamples(context.sk0.Get(), context.sk1.Get(), crp)
	}

	aggregatedSamples := make([][][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		aggregatedSamples[i] = EkgProtocol.Aggregate(context.sk1.Get(), samples, crp)
	}

	keySwitched := make([][][]*ring.Poly, params.parties)

	sum := EkgProtocol.Sum(aggregatedSamples)
	for i := uint64(0); i < params.parties; i++ {
		keySwitched[i] = EkgProtocol.KeySwitch(context.sk0.Get(), context.sk1.Get(), sum)
	}

	//EKG_V2_Round_0
	b.Run(fmt.Sprintf("EKG_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.GenSamples(context.sk0.Get(), context.sk1.Get(), crp)
		}
	})

	//EKG_V2_Round_1
	b.Run(fmt.Sprintf("EKG_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.Aggregate(context.sk1.Get(), samples, crp)
		}
	})

	//EKG_V2_Round_2
	b.Run(fmt.Sprintf("EKG_Round2"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.KeySwitch(context.sk1.Get(), context.sk1.Get(), EkgProtocol.Sum(aggregatedSamples))
		}
	})

	//EKG_V2_Round_3
	b.Run(fmt.Sprintf("EKG_Round3"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.ComputeEVK(keySwitched, sum)
		}
	})
}

func benchEKGNaive(params benchParams, context *benchContext, b *testing.B) {
	// EKG_Naive
	ekgV2Naive := NewEkgProtocolNaive(context.ckkscontext.ContextKeys(), params.bdc)

	// [nParties][CrtDecomp][WDecomp][2]
	samples := make([][][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		samples[i] = ekgV2Naive.GenSamples(context.sk0.Get(), context.pk0.Get())
	}

	aggregatedSamples := make([][][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		aggregatedSamples[i] = ekgV2Naive.Aggregate(context.sk0.Get(), context.pk0.Get(), samples)
	}

	//EKG_V2_Naive_Round_0
	b.Run(fmt.Sprintf("EKG_Naive_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.GenSamples(context.sk0.Get(), context.pk1.Get())
		}
	})

	//EKG_V2_Naive_Round_1
	b.Run(fmt.Sprintf("EKG_Naive_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.Aggregate(context.sk0.Get(), context.pk1.Get(), samples)
		}
	})

	//EKG_V2_Naive_Round_2
	b.Run(fmt.Sprintf("EKG_Naive_Round2"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.Finalize(aggregatedSamples)
		}
	})
}

func benchCKG(params benchParams, context *benchContext, b *testing.B) {

	//CKG
	ckgInstance := NewCKG(context.ckkscontext.ContextKeys(), context.cprng.Clock())
	ckgInstance.GenShare(context.sk0.Get())

	shares := make([]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		shares[i] = ckgInstance.GetShare()
	}

	// CKG_Round_0
	b.Run(fmt.Sprintf("CKG_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ckgInstance.GenShare(context.sk0.Get())
		}
	})

	// CKG_Round_1
	b.Run(fmt.Sprintf("CKG_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ckgInstance.AggregateShares(shares)
			//ckgInstance.Finalize()
		}
	})
}

func benchCKS(params benchParams, context *benchContext, b *testing.B) {
	//CKS

	cksInstance := NewCKS(context.sk0.Get(), context.sk1.Get(), context.ckkscontext.ContextKeys(), params.sigmaSmudging)

	ciphertext := context.ckkscontext.NewRandomCiphertext(1, context.ckkscontext.Levels()-1, context.ckkscontext.Scale())

	hi := make([]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		hi[i] = cksInstance.KeySwitch(ciphertext.Value()[1])
	}

	// CKS_Round_0
	b.Run(fmt.Sprintf("CKS_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cksInstance.KeySwitch(ciphertext.Value()[1])
		}
	})

	// CKS_Round_1
	b.Run(fmt.Sprintf("CKS_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cksInstance.Aggregate(ciphertext.Value()[0], hi)
		}
	})
}

func benchPCKS(params benchParams, context *benchContext, b *testing.B) {
	//CKS_Trustless
	pcks := NewPCKS(context.sk0.Get(), context.pk1.Get(), context.ckkscontext.ContextKeys(), params.sigmaSmudging)

	ciphertext := context.ckkscontext.NewRandomCiphertext(1, context.ckkscontext.Levels()-1, context.ckkscontext.Scale())

	hi := make([][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		hi[i] = pcks.KeySwitch(ciphertext.Value()[1])
	}

	// CKS_Trustless_Round_0
	b.Run(fmt.Sprintf("PCKS_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pcks.KeySwitch(ciphertext.Value()[1])
		}
	})

	// CKS_Trustless_Round_1
	b.Run(fmt.Sprintf("PCKS_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pcks.Aggregate(ciphertext.Value(), hi)
		}
	})

}
