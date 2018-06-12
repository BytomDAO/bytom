package pex

const (
	bucketTypeNew = 0x01
	bucketTypeOld = 0x02

	oldBucketSize      = 64
	oldBucketCount     = 64
	oldBucketsPerGroup = 4
	newBucketSize      = 64
	newBucketCount     = 256
	newBucketsPerGroup = 32

	getSelectionPercent = 23
	minGetSelection     = 32
	maxGetSelection     = 250

	needAddressThreshold    = 1000 // addresses under which the address manager will claim to need more addresses.
	maxNewBucketsPerAddress = 4    // buckets a frequently seen new address may end up in.
	numMissingDays          = 30   // days before which we assume an address has vanished
	numRetries              = 3    // tries without a single success before we assume an address is bad.
	maxFailures             = 10   // max failures we will accept without a success before considering an address bad.
	minBadDays              = 7    // days since the last success before we will consider evicting an address.
)
