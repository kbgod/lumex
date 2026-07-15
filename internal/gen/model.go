package gen

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type field struct {
	GoName    string
	GoType    string
	JSONName  string
	OmitEmpty bool
	Comment   string
}

type typeDecl struct {
	Name   string
	Doc    string
	Fields []field // nil → empty struct
}

type enumConst struct {
	Name  string
	Value string
}

type enumDecl struct {
	Name   string
	Owner  string
	Field  string
	Consts []enumConst
}

type unionVariant struct {
	Type      string // concrete Go type implementing the union
	DiscValue string // discriminator literal, e.g. "photo"
}

type unionInfo struct {
	Name     string
	Doc      string
	Subtypes []string       // every concrete type listed under the union
	Variants []unionVariant // subtypes with a detected discriminator value
	DiscJSON string         // discriminator JSON key ("type"/"status"/"source"); "" if none
	DiscEnum string         // Go enum type for the discriminator; "" if none
	Decoder  string         // name of the generated decode function; "" if not decodable
	Special  string         // "" normal, "maybeinaccessible" for the date-based union

	AllowsString bool // the union may also be a bare JSON string (RichText)
	AllowsArray  bool // the union may also be a JSON array of itself (RichText)
}

type methodInfo struct {
	Name       string  // API method, e.g. "sendPhoto"
	ReqType    string  // request struct, e.g. "SendPhotoRequest"
	Required   []field // required fields → positional method arguments
	OptsType   string  // "SendPhotoOpts" when the request has optional fields, else ""
	ReturnType string  // Go return type, e.g. "*Message", "bool", "[]Update", "ChatMember"
	Decoder    string  // union decoder function when ReturnType is a union interface
	MsgOrBool  bool    // API returns "Message or True" → signature is (*Message, bool, error)
}

type generator struct {
	pkg         string
	origin      string
	enableEnums bool

	sections map[string]string // type/method name → its HTML section

	decls    map[string]*typeDecl
	order    []string
	requests []*typeDecl
	methods  []methodInfo

	enums     map[string]*enumDecl
	enumOrder []string

	unionInfos     map[string]*unionInfo
	unionOrder     []string
	unions         map[string]bool   // interface-union names (field type = interface)
	emptyStruct    map[string]bool   // tableless types with no subtypes
	variantPrimary map[string]string // variant → its primary (defining) union
	sharedVariant  map[string]int    // variant → number of unions that list it
	decodable      map[string]string // union name → decoder function name

	isVariant    map[string]bool // type is a subtype of some union
	fileCarrying map[string]bool // type transitively contains an *InputFile

	typeNames  map[string]bool
	referenced map[string]bool

	version string
}

func newGenerator(pkg, origin string) *generator {
	return &generator{
		pkg:            pkg,
		origin:         origin,
		sections:       map[string]string{},
		decls:          map[string]*typeDecl{},
		enums:          map[string]*enumDecl{},
		unionInfos:     map[string]*unionInfo{},
		unions:         map[string]bool{},
		emptyStruct:    map[string]bool{},
		variantPrimary: map[string]string{},
		decodable:      map[string]string{},
		typeNames:      map[string]bool{},
		referenced:     map[string]bool{},
	}
}
