package zkCrypto

import (
	"crypto/rand"
	"crypto/sha256"
	"math"
	"math/big"
)

type Generator struct {
	N    uint64
	M    uint64
	VecG []ECPoint
	VecH []ECPoint
}

type GeneratorView struct {
	G *ECPoint
	H *ECPoint
}

const (
	numBits     uint64 = 64
	numVals     uint64 = 1
	rootNumBits uint64 = 6
)

var ZKGen Generator
var ZKView GeneratorView

func BPInit() {
	ZKGen = NewGen(numBits, numVals)
	fillVecs()
}

// NewGen generates a new chain of ECPoints that are linked by consecutive hashes
func NewGen(n, m uint64) Generator {
	BPGen := Generator{}

	BPGen.N = n
	BPGen.M = m

	u1, _ := rand.Int(rand.Reader, ZKCurve.N)
	u2, _ := rand.Int(rand.Reader, ZKCurve.N)
	VecG := genChain(n, m, u1.Bytes())
	VecH := genChain(n, m, u2.Bytes())

	BPGen.VecG = VecG
	BPGen.VecH = VecH

	return BPGen
}

func genChain(n, m uint64, initBytes []byte) []ECPoint {
	vec := make([]ECPoint, n*m)

	hasher := sha256.New()
	hasher.Write(initBytes)
	temp := new(big.Int).SetBytes(hasher.Sum(nil))
	temp.Mod(temp, ZKCurve.N)
	vec[0].X, vec[0].Y = ZKCurve.C.ScalarBaseMult(temp.Bytes())

	for ii := uint64(1); ii < n*m; ii++ {
		hasher.Write(vec[ii-1].Bytes())
		temp = new(big.Int).SetBytes(hasher.Sum(nil))
		temp.Mod(temp, ZKCurve.N)
		vec[ii].X, vec[ii].Y = ZKCurve.C.ScalarBaseMult(temp.Bytes())

		if !ZKCurve.C.IsOnCurve(vec[ii].X, vec[ii].Y) {
			Dprintf("Some is really wrong... \n")
		}
	}

	return vec
}

func (g Generator) Share(j uint64) GeneratorView {
	return GeneratorView{&g.VecG[j], &g.VecH[j]}
}

//======== Perliminary stuff

var PowsOf2 []*big.Int
var ZeroVec []*big.Int
var OnesVec []*big.Int
var RandVec []*big.Int

func fillVecs() {
	ZeroVec, OnesVec, PowsOf2, RandVec = make([]*big.Int, numBits), make([]*big.Int, numBits),
		make([]*big.Int, numBits), make([]*big.Int, numBits)
	for ii := int64(0); ii < int64(numBits); ii++ {

		// Probably can save space here
		ZeroVec[ii] = big.NewInt(0)
		OnesVec[ii] = big.NewInt(1)
		PowsOf2[ii] = new(big.Int).Exp(big.NewInt(2), big.NewInt(ii), ZKCurve.N)
	}
}

// binaryDecomp takes in a value and generates an binary composition array that
// summed up will equal value
// could probably optimize this somehow since its only 0/1
func binaryDecomp(value *big.Int) []*big.Int {

	result := make([]*big.Int, numBits)

	// GENERATES BIG ENDIAN
	for ii := 0; ii < int(numBits); ii++ {
		result[ii] = big.NewInt(0)
		value.QuoRem(value, big.NewInt(2), result[ii])
	}

	return result
}

func dotProd(x, y []*big.Int) *big.Int {

	if len(x) != len(y) {
		return big.NewInt(0)
	}

	acc := big.NewInt(0)

	for ii := 0; ii < len(x); ii++ {
		acc.Add(new(big.Int).Mul(x[ii], y[ii]), big.NewInt(0))
	}
	return acc
}

// x[] cannot contain 0 or 1 in any entry?
func ecDotProd(x []*big.Int, y []ECPoint) ECPoint {
	if len(x) != len(y) {
		return ZKCurve.Zero()
	}

	acc := ZKCurve.Zero()
	acc.X, acc.Y = ZKCurve.C.ScalarMult(y[0].X, y[0].Y, x[0].Bytes())
	temp := ZKCurve.Zero()

	for ii := 1; ii < len(x); ii++ {
		temp.X, temp.Y = ZKCurve.C.ScalarMult(y[ii].X, y[ii].Y, x[ii].Bytes())
		acc = acc.Add(temp)
	}
	return acc
}

// COM(vec1, vec2, u) -> <vec1, G> + <vec2, H> + uB', where B' is the ped commit blinder, G and H are chain vecots
func vecPedComm(a []*big.Int, G []ECPoint, H []ECPoint) (ECPoint, *big.Int) {
	if len(a) != len(G) || len(a) != len(H) {
		return ZKCurve.Zero(), big.NewInt(0)
	}

	randomness, _ := rand.Int(rand.Reader, ZKCurve.N)
	res := ecDotProd(a, G).Add(ecDotProd(a, H))
	temp := ZKCurve.Zero()
	temp.X, temp.Y = ZKCurve.C.ScalarBaseMult(randomness.Bytes())
	res = res.Add(temp)

	return res, randomness

}

func vecMult(x, y []*big.Int) []*big.Int {

	if len(x) != len(y) {
		return []*big.Int{}
	}

	res := make([]*big.Int, len(x))

	for ii := 0; ii < len(x); ii++ {
		res[ii] = new(big.Int).Mul(x[ii], y[ii]) // res is not declared yet so we need assignment statement
	}
	return res
}

func vecAdd(x, y []*big.Int) []*big.Int {
	if len(x) != len(y) {
		return []*big.Int{}
	}

	res := make([]*big.Int, len(x))

	for ii := 0; ii < len(x); ii++ {
		res[ii] = new(big.Int).Add(x[ii], y[ii]) // res is not declared yet so we need assignment statement
	}
	return res
}

func vecAddEC(G []ECPoint, H []ECPoint) []ECPoint {
	if len(G) != len(H) {
		return []ECPoint{}
	}

	res := make([]ECPoint, len(G))

	for ii := 0; ii < len(G); ii++ {
		res[ii].X, res[ii].Y = ZKCurve.C.Add(G[ii].X, G[ii].Y, H[ii].X, H[ii].Y) // res is not declared yet so we need assignment statement
	}
	return res

}

func vecSub(x, y []*big.Int) []*big.Int {
	if len(x) != len(y) {
		return []*big.Int{}
	}

	res := make([]*big.Int, len(x))
	for ii := 0; ii < len(x); ii++ {
		res[ii] = new(big.Int).Sub(x[ii], y[ii]) // res is not declared yet so we need assignment statement
	}
	return res
}

func splitVec(x []*big.Int) ([]*big.Int, []*big.Int) {

	if len(x)%2 != 0 {
		Dprintf("splitVec:\n - input arrays are not multiple of 2\n")
		return []*big.Int{}, []*big.Int{}
	}

	left, right := make([]*big.Int, len(x)/2), make([]*big.Int, len(x)/2)
	left, right = x[0:len(x)/2], x[len(x)/2:len(x)] // WHY DOES IT HAVE TO BE SLICES AND NOT INDEXES

	return left, right
}

func splitVecEC(x []ECPoint) ([]ECPoint, []ECPoint) {

	if len(x)%2 != 0 {
		return []ECPoint{}, []ECPoint{}
	}

	return x[0 : len(x)/2], x[len(x)/2 : len(x)]
}

func genVec(x *big.Int) []*big.Int {
	res := make([]*big.Int, numBits)

	for ii := int64(0); ii < int64(numBits); ii++ {
		res[ii] = new(big.Int).Exp(x, big.NewInt(ii), ZKCurve.N)
	}
	return res
}

func scalar(x []*big.Int, y *big.Int) []*big.Int {
	if len(x)%2 != 0 && len(x) != 1 {
		return []*big.Int{}
	}

	res := make([]*big.Int, len(x))
	for ii := 0; ii < len(x); ii++ {
		res[ii] = new(big.Int).Mul(x[ii], y)
	}
	return res

}

func scalarEC(x *big.Int, G []ECPoint) []ECPoint {
	if len(G)%2 != 0 && len(G) != 1 {
		return []ECPoint{}
	}

	res := make([]ECPoint, len(G))
	for ii := 0; ii < len(G); ii++ {
		res[ii].X, res[ii].Y = ZKCurve.C.ScalarMult(G[ii].X, G[ii].Y, x.Bytes())
	}
	return res

}

// // t(x) = ax^2 bx + c
// type TwoDegPoly struct {
// 	A []*big.Int
// 	B []*big.Int
// 	C []*big.Int
// }

// // evalualte t(x)
// func (p *TwoDegPoly) EvalAt(x *big.Int) []*big.Int {
// 	res := make([]*big.Int, numBits)
// 	// I know this looks nasty
// 	for ii := uint64(0); ii < numBits; ii++ {
// 		res[ii] = new(big.Int).Add(new(big.Int).Add(new(big.Int).Mul(p.b[ii], x), p.c[ii]), new(big.Int).Mul(p.a[ii], x).Mul(x, big.NewInt(1)))
// 	}

// 	return res
// }

//========== INNER PRODUCT PROOF =========

type InProdProof struct {
	A        *big.Int
	B        *big.Int
	U        []*big.Int
	UInv     []*big.Int
	P        ECPoint
	Q        ECPoint
	LeftVec  []ECPoint
	RightVec []ECPoint
}

/*
	Inner Product Proof:

	REFERANCE : https://doc-internal.dalek.rs/ristretto_bulletproofs/inner_product_proof/index.html

	We want to prove that c is the inner product (dot product) of a and b
	without sending 2n bits but insteading sending log(n) bits

	Public: G, H, P', Q, a, b, where:
	 - a and b are scalar vectors
	 - G and H are ECPoint vectors
	 - P' and Q are ECPoints

	c = <a, b>
	P = <a,G> + <b,H>
	w is random num

	P' = cwB + P where B is the G of the ped commitment
	Q = wB



*/

func InProdProve(a, b []*big.Int, G, H []ECPoint) (InProdProof, bool) {

	if len(a)%2 != 0 || len(a) != len(b) || len(a) != len(G) || len(a) != len(H) {
		Dprintf("InProdProof:\n - lengths of arrays do not agree/not multiple of 2\n")
		return InProdProof{}, false
	}

	k := rootNumBits

	proof := InProdProof{big.NewInt(0),
		big.NewInt(0),
		make([]*big.Int, k),
		make([]*big.Int, k),
		ZKCurve.Zero(),
		ZKCurve.Zero(),
		make([]ECPoint, k),
		make([]ECPoint, k)}

	// Commitments we want to prove
	temp1 := ecDotProd(a, G)
	temp2 := ecDotProd(a, H)
	P := temp1.Add(temp2)
	c := dotProd(a, b)

	// Blinding factor for c
	w, _ := rand.Int(rand.Reader, ZKCurve.N)

	// Multiple by the G of pedersen commitment, which is BaseMult
	QX, QY := ZKCurve.C.ScalarBaseMult(w.Bytes())
	Q := ECPoint{QX, QY}
	proof.Q = Q

	// c * wB
	temp1X, temp1Y := ZKCurve.C.ScalarMult(QX, QY, c.Bytes())
	// P' = P + cwB, public commitment used for inner product proof
	P2 := P.Add(ECPoint{temp1X, temp1Y})
	proof.P = P2

	hasher := sha256.New()

	for ii := uint64(0); ii < k; ii++ {
		// split the vectors for reduction later
		aL, aR := splitVec(a)
		bL, bR := splitVec(b)
		GL, GR := splitVecEC(G)
		HL, HR := splitVecEC(H)

		// Prover calcualtes L and R
		proof.LeftVec[ii] = ecDotProd(aL, GR).Add(ecDotProd(bR, HL).Add(Q.Mult(dotProd(aL, bR))))
		proof.RightVec[ii] = ecDotProd(aR, GL).Add(ecDotProd(bL, HR).Add(Q.Mult(dotProd(aR, bL))))
		// FS-Transform to make this non interactive is to write in each L and R into the buffer
		// stringing these consecutive hashes locks in each value of L and R to the previous ones
		hasher.Write(proof.LeftVec[ii].Bytes())
		hasher.Write(proof.RightVec[ii].Bytes())
		u := new(big.Int).SetBytes(hasher.Sum(nil))
		uinv := new(big.Int).ModInverse(u, ZKCurve.N)
		proof.U[ii] = new(big.Int).Mul(u, u)          // we need it squred for verification
		proof.UInv[ii] = new(big.Int).Mul(uinv, uinv) //need squared for verification

		// reduce vectors by half
		// a, b are computed by verifier only
		a = vecAdd(scalar(aL, u), scalar(aR, uinv))
		b = vecAdd(scalar(bR, u), scalar(bL, uinv))
		// G, H are computed by both parites
		G = vecAddEC(scalarEC(u, GR), scalarEC(uinv, GL))
		H = vecAddEC(scalarEC(u, HL), scalarEC(uinv, HR))

	}

	if len(a) != 1 || len(a) != len(b) {
		Dprintf("InProdProof:\n - len(a) is not 1 and/or len(a) != len(b)\n")
		Dprintf(" - Proof failed to generate\n")
		Dprintf(" - a: %v\n - b: %v\n", a, b)
		return InProdProof{}, false
	}

	proof.A = a[0]
	proof.B = b[0]

	return proof, true
}

func InProdVerify(G, H []ECPoint, proof InProdProof) bool {

	s := make([]*big.Int, numBits)
	sInv := make([]*big.Int, numBits)
	for ii := 1; ii < int(numBits)+1; ii++ { //cant do log(0)... so gotta start at 1 and compensate for it later
		lgI := uint64(math.Log2(float64(ii)))
		k := 1 << lgI
		if ii-k < 6 && k != 64 {
			s[ii-1] = new(big.Int).Mul(proof.UInv[ii-k], proof.U[rootNumBits-1-lgI])
		} else {
			s[ii-1] = new(big.Int).Mul(s[ii-k], proof.U[rootNumBits-lgI])
		}
		sInv[int(numBits)-ii] = s[ii-1] // reverse order of s provides multiplicative inverses
	}

	// temp1 = <a * s, G>
	// temp2 = <b / s, H>
	// temp3 = abQ
	// temp4 = - SUM(uL + uinvR) for j = 0 -> k-1

	temp1 := ecDotProd(scalar(s, proof.A), G)
	temp2 := ecDotProd(scalar(sInv, proof.B), H)
	temp3 := proof.Q.Mult(proof.A.Mul(proof.A, proof.B))

	temps := temp1.Add(temp2.Add(temp3))

	sumTemp := ZKCurve.Zero()

	for ii, _ := range proof.U {
		sumTemp = sumTemp.Add(proof.LeftVec[ii].Mult(proof.U[ii]).Add(proof.RightVec[ii].Mult(proof.UInv[ii])))
	}

	if !proof.P.Equal(temps.Add(sumTemp)) {
		return false
	}

	return true
}

/* RANDOM STUFF I WROTE FOR SOME REASON

Here we want to prove the following three statements within one inner-product:
	- <a_L, 2^n> = v
	- a_L .* a_R = 0
	- (a_L - 1) - a_R = 0

	The above three statements combine into:
	- z^2 * v + [(z - z^2) <1, y^n> - z^3 <1,2^n>] =
		<a_L, -z1,y^n .* (a_R + 1) + z^2 2^n>

	where:
	- a_L = a = binary composition of v
	- a_R = a - 1
	- 0, 1 	are vecotrs of 0 and 1
	- y^n 	is a challenge vector of [0, 1, y, y^2..., y^n-1]
	- z 	is a challenge scalar
	- the bracketed [] expression will be represned as delta:
		- delta = (z - z^2) <1, y^n> - z^3 <1,2^n>

	The left side of this expression can be computed publically
	The right side can only be done by the prover


aL := binaryDecomp(x)
aR := vecSub(aL, OnesVec)

if dotProd(aL, PowsOf2).Cmp(x) != 0 {
	Dprintf("InProdProve:\n - Was not able to generate binaryDecomp")
	return InProdProof{}
}

if dotProd(aL, aR).Cmp(big.NewInt(0)) != 0 {
	Dprintf("InProdProve:\n - aL .* aR did not result in 0")
	return InProdProof{}
}

// Challenge vector and challenge scalar
// y, _ := rand.Int(rand.Reader, ZKCurve.N)
// yVec := genVec(y)
// z, _ := rand.Int(rand.Reader, ZKCurve.N)


// delta = (z - z^2) <1, y^n> - z^3 <1,2^n>
// temp1 := scalarMult(OnesVec, new(big.Int).Sub(z, new(big.Int).Mul(z, z)))
// temp2 := scalarMult(OnesVec, new(big.Int).Exp(z, big.NewInt(3), ZKCurve.N))
// delta := new(big.Int).Sub(dotProd(temp1, yVec), dotProd(temp2, PowsOf2))

c := dotProd(aL, aR)



*/
