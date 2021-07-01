package initblocks

const (
	OutputCntPerTx = 3
	TxCntPerBlock  = 3
)

type AssetTotal struct {
	Asset  string
	Amount uint64
}

// demo
var assetTotals = []AssetTotal{
	{Asset: "f58248ab687d8ee428f9be5f4ec4a5ff690ca5dd7c28a80efdfd7f4a816894f6", Amount: 210000000000000000},
	{Asset: "f7612359fd97eb16b806a3cdb7fce289acab0dd40d4240980ce6a3e4b515f6fd", Amount: 548997812901981},
	{Asset: "f7c41b58fc6b4c006fe2baf68f73ff3b26218f1f6bee5481e370681d9c19da6b", Amount: 99999999899999990},
	{Asset: "f99ee16a303949ed9e6580cb6dd5105ca5e3c5f888ebe0a4efb9a54aafd59b9b", Amount: 100000000000000},
	{Asset: "f9d9e23f34e72a72976742e0e2a80b4715a3481ec9c8450e6ae853ea86b26e29", Amount: 100000000},
	{Asset: "fbbad45dac8aa24118c5a1249fedf566e438a9415b5277c7f870edbd9988081d", Amount: 410615614295013},
	{Asset: "fc49b693a094a975a15a4b21ebe118d4a6c5754e0cbc341baf687f56064a9074", Amount: 14000000000},
	{Asset: "ff0579aea42e263ce1bdfc618b81afbc42623651daf184d4fcdf7b41baed52fc", Amount: 21000000000000},
	{Asset: "ff8f19c161a6389ca3ffac3ea6f77fe7de13756d4339d231737d86a3988cfe62", Amount: 4500000000000000},
	{Asset: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", Amount: 167959664788809797},
}

type AddressBalance struct {
	Address string
	Balance uint64
}

// demo
var asset2distributions = map[string][]AddressBalance{
	"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff": {
		{Address: "bm1qzzy4dx4zlr2ve5p9smv8xj7z7lp49eqvzjyqel", Balance: 29963022217},
		{Address: "bm1qzzxrpd5ehedxla7rtlhaahjd242syd5a6myjdx", Balance: 1710990},
		{Address: "bm1qzzwv8nnn0vpju5s067j9s8tq5g2lznr67mflc9", Balance: 1300000000},
		{Address: "bm1qzztfk0z8ejc3q2ez2skcwcsup7hxsw2vy9c6ch", Balance: 300000000},
		{Address: "bm1qzztaltajqceedqfucxf0n0hmuuay587t02hf9h", Balance: 106805795},
		{Address: "bm1qzzt9pgc5ygvmemhdnars3ccrjvzge4lmarrlhp", Balance: 25300753},
		{Address: "bm1qzzqwdz0uwftvex00ty6f6wt5ddc9xsusjmvqm3", Balance: 297959600},
		{Address: "bm1qzzpy270ywv9f0f289wyejxcrcejyh0mwuqr2m3", Balance: 81764287},
		{Address: "bm1qzzpcylqjdppz9t7xnpz2c8ztnwjmd8hcry6qpl", Balance: 5102000},
		{Address: "bm1qzzk8cpwdphyz3l0epd44uvtq5jnuckwavgcuup", Balance: 4102000},
		{Address: "bm1qzzhn5dcun3lsfvgsr4rhjqe762xqrzv5wntpqp", Balance: 8653000},
		{Address: "bm1qzzhmypa9gmczmuq29ck68ep633dexn2n86t9rc", Balance: 9102000},
		{Address: "bm1qzzgeu6nytljtr8ku244ddsf7y09fshj3wtgsca", Balance: 343077593},
		{Address: "bm1qzzfpfsauu8vzeapyjl5fpspaaeg0cxfntnemr3", Balance: 369522470000},
		{Address: "bm1qzze78evmmgx7t0r9f7pzz0dxp7kra96xj088ux", Balance: 30700000000},
		{Address: "bm1qzze6ag9eg5s4ser8jjdvcm5g06ezqz5fzvcm77", Balance: 94100000},
		{Address: "bm1qzzcmk0l3kultf85x0r9xngwetpvutnh3z9vsjv", Balance: 20898824280},
		{Address: "bm1qzzap6gtlhgupaslvfv69jp2qdgp90kacdzpu78", Balance: 45671000},
		{Address: "bm1qzza9t8zcly09scnc98rr396jc37agtdu7pnut5", Balance: 957175000},
		{Address: "bm1qzz8p284kf6kfc57cmxs9r2mt30tz3tskyygv0g", Balance: 3959000},
		{Address: "bm1qzz5lk36yarl4hp4fj3lx7s65wyy5d38j68n7nt", Balance: 62700642},
		{Address: "bm1qzz50qsn90np779w885dp9dl7cppqxwzt49sqyg", Balance: 1825000000000},
		{Address: "bm1qzz4xscgfc35agxxd7ksdjav82srcck28kz3cxg", Balance: 6857000},
	},
	"f58248ab687d8ee428f9be5f4ec4a5ff690ca5dd7c28a80efdfd7f4a816894f6": {
		{Address: "bm1qyst3ua85grmzsjfvcc9pwfvrznl36cgruspzxs", Balance: 380000000000},
		{Address: "bm1qy3jmzxp34darp5mlwpm774pt2zn2mcqz8q40p3", Balance: 14100000000000},
		{Address: "bm1qxdte9929l6gydkvg6w4j6hck5kwsshd833y0l5", Balance: 4200000000000},
		{Address: "bm1qrr4895r36csusy6p36kv8mlt6juwku52j0pn7d", Balance: 209724038300000000},
		{Address: "bm1qqd4chtqmzctg7c2cptgsakf2jghw3h707wgwrz", Balance: 19280000000000},
		{Address: "bm1qq8ylf2a9084lc84s84vy24ajy9dvpyu850ljzp", Balance: 18500000000000},
		{Address: "bm1qnzd85gwxlvy40rf8jll23y3kwps0jfugwwgws9", Balance: 4560000000000},
		{Address: "bm1qljgfl3z3d2cj59msu3knp37mlmtuef4yprzp2e", Balance: 121403700000000},
		{Address: "bm1qk5q8c7d9msyvrcfq7mqqhhtcmvp9fy8qdkja6y", Balance: 3940000000000},
		{Address: "bm1qjr62a9aa35ntvnv5mu2n3uyewcadpxmtqqcsuf", Balance: 3800000000000},
		{Address: "bm1qhdx2esyljqxvfvmamd4rcwl3v8d5yyjxe4xk2d", Balance: 5000000000000},
		{Address: "bm1qf2px9fmwpvy9zmvxyfzpkv6f87vqjf83l03w5w", Balance: 30000000000},
	},
}
