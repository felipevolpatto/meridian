package generator

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
)

// SemanticFieldType represents detected field types based on naming
type SemanticFieldType int

const (
	SemanticUnknown SemanticFieldType = iota
	SemanticID
	SemanticName
	SemanticFirstName
	SemanticLastName
	SemanticFullName
	SemanticEmail
	SemanticPhone
	SemanticAddress
	SemanticStreet
	SemanticCity
	SemanticState
	SemanticCountry
	SemanticZipCode
	SemanticPostalCode
	SemanticURL
	SemanticWebsite
	SemanticUsername
	SemanticPassword
	SemanticTitle
	SemanticDescription
	SemanticContent
	SemanticBody
	SemanticMessage
	SemanticCompany
	SemanticOrganization
	SemanticPrice
	SemanticAmount
	SemanticQuantity
	SemanticCount
	SemanticAge
	SemanticDate
	SemanticCreatedAt
	SemanticUpdatedAt
	SemanticBirthday
	SemanticImage
	SemanticAvatar
	SemanticPhoto
	SemanticColor
	SemanticStatus
	SemanticType
	SemanticCategory
	SemanticTag
	SemanticSlug
	SemanticCode
	SemanticSKU
	SemanticISBN
	SemanticLatitude
	SemanticLongitude
	SemanticCurrency
	SemanticLanguage
	SemanticTimezone
	SemanticIPAddress
	SemanticUserAgent
	SemanticCreditCard
)

var semanticPatterns = map[*regexp.Regexp]SemanticFieldType{
	regexp.MustCompile(`(?i)^id$|_id$|Id$`):                             SemanticID,
	regexp.MustCompile(`(?i)^first_?name$|^given_?name$`):               SemanticFirstName,
	regexp.MustCompile(`(?i)^last_?name$|^family_?name$|^surname$`):     SemanticLastName,
	regexp.MustCompile(`(?i)^full_?name$|^display_?name$`):              SemanticFullName,
	regexp.MustCompile(`(?i)^name$`):                                    SemanticName,
	regexp.MustCompile(`(?i)^e?mail$|^email_?address$`):                 SemanticEmail,
	regexp.MustCompile(`(?i)^phone$|^phone_?number$|^mobile$|^tel$`):    SemanticPhone,
	regexp.MustCompile(`(?i)^address$|^full_?address$`):                 SemanticAddress,
	regexp.MustCompile(`(?i)^street$|^street_?address$|^line1$`):        SemanticStreet,
	regexp.MustCompile(`(?i)^city$|^town$`):                             SemanticCity,
	regexp.MustCompile(`(?i)^state$|^province$|^region$`):               SemanticState,
	regexp.MustCompile(`(?i)^country$|^nation$`):                        SemanticCountry,
	regexp.MustCompile(`(?i)^zip$|^zip_?code$`):                         SemanticZipCode,
	regexp.MustCompile(`(?i)^postal_?code$|^postcode$`):                 SemanticPostalCode,
	regexp.MustCompile(`(?i)^url$|^link$|^href$`):                       SemanticURL,
	regexp.MustCompile(`(?i)^website$|^homepage$|^site$`):               SemanticWebsite,
	regexp.MustCompile(`(?i)^user_?name$|^login$|^handle$`):             SemanticUsername,
	regexp.MustCompile(`(?i)^password$|^pass$|^pwd$|^secret$`):          SemanticPassword,
	regexp.MustCompile(`(?i)^title$|^headline$|^subject$`):              SemanticTitle,
	regexp.MustCompile(`(?i)^description$|^desc$|^summary$|^bio$`):      SemanticDescription,
	regexp.MustCompile(`(?i)^content$|^text$|^body$`):                   SemanticContent,
	regexp.MustCompile(`(?i)^message$|^comment$|^note$`):                SemanticMessage,
	regexp.MustCompile(`(?i)^company$|^business$|^employer$`):           SemanticCompany,
	regexp.MustCompile(`(?i)^organization$|^org$|^institution$`):        SemanticOrganization,
	regexp.MustCompile(`(?i)^price$|^cost$|^fee$`):                      SemanticPrice,
	regexp.MustCompile(`(?i)^amount$|^total$|^sum$|^balance$`):          SemanticAmount,
	regexp.MustCompile(`(?i)^quantity$|^qty$`):                          SemanticQuantity,
	regexp.MustCompile(`(?i)^count$|^num$|^number$`):                    SemanticCount,
	regexp.MustCompile(`(?i)^age$`):                                     SemanticAge,
	regexp.MustCompile(`(?i)^date$`):                                    SemanticDate,
	regexp.MustCompile(`(?i)^created_?at$|^creation_?date$`):            SemanticCreatedAt,
	regexp.MustCompile(`(?i)^updated_?at$|^modified_?at$|^edit_?date$`): SemanticUpdatedAt,
	regexp.MustCompile(`(?i)^birthday$|^birth_?date$|^dob$`):            SemanticBirthday,
	regexp.MustCompile(`(?i)^image$|^img$|^picture$`):                   SemanticImage,
	regexp.MustCompile(`(?i)^avatar$|^profile_?image$|^photo$`):         SemanticAvatar,
	regexp.MustCompile(`(?i)^color$|^colour$`):                          SemanticColor,
	regexp.MustCompile(`(?i)^status$`):                                  SemanticStatus,
	regexp.MustCompile(`(?i)^type$|^kind$`):                             SemanticType,
	regexp.MustCompile(`(?i)^category$|^cat$`):                          SemanticCategory,
	regexp.MustCompile(`(?i)^tag$|^label$`):                             SemanticTag,
	regexp.MustCompile(`(?i)^slug$|^permalink$`):                        SemanticSlug,
	regexp.MustCompile(`(?i)^code$`):                                    SemanticCode,
	regexp.MustCompile(`(?i)^sku$|^product_?code$`):                     SemanticSKU,
	regexp.MustCompile(`(?i)^isbn$`):                                    SemanticISBN,
	regexp.MustCompile(`(?i)^lat$|^latitude$`):                          SemanticLatitude,
	regexp.MustCompile(`(?i)^lng$|^lon$|^longitude$`):                   SemanticLongitude,
	regexp.MustCompile(`(?i)^currency$|^currency_?code$`):               SemanticCurrency,
	regexp.MustCompile(`(?i)^language$|^lang$|^locale$`):                SemanticLanguage,
	regexp.MustCompile(`(?i)^timezone$|^tz$|^time_?zone$`):              SemanticTimezone,
	regexp.MustCompile(`(?i)^ip$|^ip_?address$`):                        SemanticIPAddress,
	regexp.MustCompile(`(?i)^user_?agent$|^ua$`):                        SemanticUserAgent,
	regexp.MustCompile(`(?i)^credit_?card$|^card_?number$|^cc$`):        SemanticCreditCard,
}

// DetectSemanticType detects the semantic type of a field based on its name
func DetectSemanticType(fieldName string) SemanticFieldType {
	for pattern, semanticType := range semanticPatterns {
		if pattern.MatchString(fieldName) {
			return semanticType
		}
	}
	return SemanticUnknown
}

// GenerateBySemanticType generates data based on detected semantic type
func GenerateBySemanticType(semanticType SemanticFieldType, schema *openapi3.Schema) interface{} {
	f := faker.New()

	switch semanticType {
	case SemanticID:
		return f.UUID().V4()
	case SemanticFirstName:
		return f.Person().FirstName()
	case SemanticLastName:
		return f.Person().LastName()
	case SemanticFullName, SemanticName:
		return f.Person().Name()
	case SemanticEmail:
		return f.Internet().Email()
	case SemanticPhone:
		return f.Phone().Number()
	case SemanticAddress:
		return fmt.Sprintf("%s, %s, %s %s",
			f.Address().StreetAddress(),
			f.Address().City(),
			f.Address().State(),
			f.Address().PostCode())
	case SemanticStreet:
		return f.Address().StreetAddress()
	case SemanticCity:
		return f.Address().City()
	case SemanticState:
		return f.Address().State()
	case SemanticCountry:
		return f.Address().Country()
	case SemanticZipCode, SemanticPostalCode:
		return f.Address().PostCode()
	case SemanticURL, SemanticWebsite:
		return f.Internet().URL()
	case SemanticUsername:
		return f.Internet().User()
	case SemanticPassword:
		return f.Internet().Password()
	case SemanticTitle:
		return f.Lorem().Sentence(4)
	case SemanticDescription, SemanticContent, SemanticBody, SemanticMessage:
		return f.Lorem().Paragraph(2)
	case SemanticCompany, SemanticOrganization:
		return f.Company().Name()
	case SemanticPrice, SemanticAmount:
		return f.Float64(2, 1, 1000)
	case SemanticQuantity, SemanticCount:
		return f.IntBetween(1, 100)
	case SemanticAge:
		return f.IntBetween(18, 80)
	case SemanticDate:
		return f.Time().Time(time.Now()).Format("2006-01-02")
	case SemanticCreatedAt, SemanticUpdatedAt:
		return f.Time().Time(time.Now()).Format(time.RFC3339)
	case SemanticBirthday:
		year := f.IntBetween(1950, 2005)
		month := f.IntBetween(1, 12)
		day := f.IntBetween(1, 28)
		return fmt.Sprintf("%d-%02d-%02d", year, month, day)
	case SemanticImage, SemanticAvatar, SemanticPhoto:
		return fmt.Sprintf("https://picsum.photos/seed/%s/400/400", f.UUID().V4()[:8])
	case SemanticColor:
		return f.Color().Hex()
	case SemanticStatus:
		statuses := []string{"active", "inactive", "pending", "completed", "cancelled"}
		return statuses[rand.Intn(len(statuses))]
	case SemanticType, SemanticCategory:
		return f.Lorem().Word()
	case SemanticTag:
		return f.Lorem().Word()
	case SemanticSlug:
		return strings.ToLower(strings.ReplaceAll(f.Lorem().Sentence(3), " ", "-"))
	case SemanticCode:
		return strings.ToUpper(f.Lorem().Word())[:3] + "-" + strconv.Itoa(f.IntBetween(1000, 9999))
	case SemanticSKU:
		return fmt.Sprintf("SKU-%s-%04d", strings.ToUpper(f.Lorem().Word())[:3], f.IntBetween(1, 9999))
	case SemanticISBN:
		return fmt.Sprintf("978-%d-%d-%d-%d",
			f.IntBetween(0, 9),
			f.IntBetween(1000, 9999),
			f.IntBetween(1000, 9999),
			f.IntBetween(0, 9))
	case SemanticLatitude:
		return f.Float64(6, -90, 90)
	case SemanticLongitude:
		return f.Float64(6, -180, 180)
	case SemanticCurrency:
		currencies := []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "BRL"}
		return currencies[rand.Intn(len(currencies))]
	case SemanticLanguage:
		languages := []string{"en", "es", "fr", "de", "pt", "it", "ja", "zh", "ko", "ru"}
		return languages[rand.Intn(len(languages))]
	case SemanticTimezone:
		timezones := []string{
			"America/New_York", "America/Los_Angeles", "Europe/London",
			"Europe/Paris", "Asia/Tokyo", "Asia/Shanghai", "Australia/Sydney",
		}
		return timezones[rand.Intn(len(timezones))]
	case SemanticIPAddress:
		return f.Internet().Ipv4()
	case SemanticUserAgent:
		userAgents := []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
		}
		return userAgents[rand.Intn(len(userAgents))]
	case SemanticCreditCard:
		return f.Payment().CreditCardNumber()
	default:
		return nil
	}
}

// GenerateFromPattern generates a string that matches the given regex pattern
func GenerateFromPattern(pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("empty pattern")
	}

	gen := &patternGenerator{
		faker: faker.New(),
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return gen.generate(pattern)
}

type patternGenerator struct {
	faker faker.Faker
	rand  *rand.Rand
}

func (g *patternGenerator) generate(pattern string) (string, error) {
	var result strings.Builder
	i := 0

	for i < len(pattern) {
		switch pattern[i] {
		case '\\':
			if i+1 < len(pattern) {
				escapeChar := pattern[i+1]
				char, advance := g.handleEscape(pattern[i+1:])
				nextPos := i + 1 + advance
				// Check for quantifier after escape sequence
				if nextPos < len(pattern) && isQuantifier(pattern[nextPos]) {
					quantified, qAdvance := g.applyQuantifierWithGenerator(func() string {
						c, _ := g.handleEscape(string(escapeChar))
						return c
					}, pattern[nextPos:])
					result.WriteString(quantified)
					i = nextPos + qAdvance
				} else {
					result.WriteString(char)
					i = nextPos
				}
			} else {
				result.WriteByte('\\')
				i++
			}
		case '[':
			end := strings.Index(pattern[i:], "]")
			if end == -1 {
				return "", fmt.Errorf("unclosed character class at position %d", i)
			}
			classContent := pattern[i+1 : i+end]
			char := g.handleCharClass(classContent)
			// Check for quantifier after character class
			nextPos := i + end + 1
			if nextPos < len(pattern) && isQuantifier(pattern[nextPos]) {
				quantified, advance := g.applyQuantifierWithGenerator(func() string {
					return g.handleCharClass(classContent)
				}, pattern[nextPos:])
				result.WriteString(quantified)
				i = nextPos + advance
			} else {
				result.WriteString(char)
				i = nextPos
			}
		case '(':
			end := g.findMatchingParen(pattern[i:])
			if end == -1 {
				return "", fmt.Errorf("unclosed group at position %d", i)
			}
			groupContent := pattern[i+1 : i+end]
			generated, err := g.handleGroup(groupContent)
			if err != nil {
				return "", err
			}
			// Check for quantifier after group
			if i+end+1 < len(pattern) {
				quantified, advance := g.applyQuantifier(generated, pattern[i+end+1:])
				result.WriteString(quantified)
				i += end + 1 + advance
			} else {
				result.WriteString(generated)
				i += end + 1
			}
		case '.':
			result.WriteByte(g.randomPrintable())
			i++
		case '^', '$':
			i++
		case '{', '}', '+', '*', '?':
			i++
		default:
			// Check for quantifier
			if i+1 < len(pattern) && isQuantifier(pattern[i+1]) {
				char := string(pattern[i])
				quantified, advance := g.applyQuantifier(char, pattern[i+1:])
				result.WriteString(quantified)
				i += 1 + advance
			} else {
				result.WriteByte(pattern[i])
				i++
			}
		}
	}

	return result.String(), nil
}

func (g *patternGenerator) handleEscape(s string) (string, int) {
	if len(s) == 0 {
		return "", 0
	}

	switch s[0] {
	case 'd':
		return string('0' + byte(g.rand.Intn(10))), 1
	case 'D':
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*"
		return string(chars[g.rand.Intn(len(chars))]), 1
	case 'w':
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
		return string(chars[g.rand.Intn(len(chars))]), 1
	case 'W':
		chars := "!@#$%^&*()+-=[]{}|;':\",./<>?"
		return string(chars[g.rand.Intn(len(chars))]), 1
	case 's':
		return " ", 1
	case 'S':
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		return string(chars[g.rand.Intn(len(chars))]), 1
	case 'n':
		return "\n", 1
	case 't':
		return "\t", 1
	case 'r':
		return "\r", 1
	default:
		return string(s[0]), 1
	}
}

func (g *patternGenerator) handleCharClass(class string) string {
	if len(class) == 0 {
		return ""
	}

	negated := class[0] == '^'
	if negated {
		class = class[1:]
	}

	var chars []rune
	i := 0
	for i < len(class) {
		if i+2 < len(class) && class[i+1] == '-' {
			start := rune(class[i])
			end := rune(class[i+2])
			for c := start; c <= end; c++ {
				chars = append(chars, c)
			}
			i += 3
		} else {
			chars = append(chars, rune(class[i]))
			i++
		}
	}

	if len(chars) == 0 {
		return ""
	}

	if negated {
		allChars := make([]rune, 0)
		for c := rune(32); c < 127; c++ {
			found := false
			for _, excluded := range chars {
				if c == excluded {
					found = true
					break
				}
			}
			if !found {
				allChars = append(allChars, c)
			}
		}
		if len(allChars) > 0 {
			return string(allChars[g.rand.Intn(len(allChars))])
		}
		return ""
	}

	return string(chars[g.rand.Intn(len(chars))])
}

func (g *patternGenerator) handleGroup(content string) (string, error) {
	if strings.HasPrefix(content, "?:") {
		content = content[2:]
	}

	if strings.Contains(content, "|") {
		alternatives := strings.Split(content, "|")
		choice := alternatives[g.rand.Intn(len(alternatives))]
		return g.generate(choice)
	}

	return g.generate(content)
}

func (g *patternGenerator) findMatchingParen(s string) int {
	depth := 0
	for i, c := range s {
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func (g *patternGenerator) applyQuantifier(base string, quantifier string) (string, int) {
	if len(quantifier) == 0 {
		return base, 0
	}

	switch quantifier[0] {
	case '*':
		count := g.rand.Intn(5)
		return strings.Repeat(base, count), 1
	case '+':
		count := 1 + g.rand.Intn(4)
		return strings.Repeat(base, count), 1
	case '?':
		if g.rand.Intn(2) == 0 {
			return "", 1
		}
		return base, 1
	case '{':
		end := strings.Index(quantifier, "}")
		if end == -1 {
			return base, 0
		}
		rangeStr := quantifier[1:end]
		min, max := g.parseRange(rangeStr)
		count := min + g.rand.Intn(max-min+1)
		return strings.Repeat(base, count), end + 1
	default:
		return base, 0
	}
}

func (g *patternGenerator) applyQuantifierWithGenerator(gen func() string, quantifier string) (string, int) {
	if len(quantifier) == 0 {
		return gen(), 0
	}

	var count int
	var advance int

	switch quantifier[0] {
	case '*':
		count = g.rand.Intn(5)
		advance = 1
	case '+':
		count = 1 + g.rand.Intn(4)
		advance = 1
	case '?':
		if g.rand.Intn(2) == 0 {
			return "", 1
		}
		return gen(), 1
	case '{':
		end := strings.Index(quantifier, "}")
		if end == -1 {
			return gen(), 0
		}
		rangeStr := quantifier[1:end]
		min, max := g.parseRange(rangeStr)
		count = min + g.rand.Intn(max-min+1)
		advance = end + 1
	default:
		return gen(), 0
	}

	var result strings.Builder
	for i := 0; i < count; i++ {
		result.WriteString(gen())
	}
	return result.String(), advance
}

func (g *patternGenerator) parseRange(rangeStr string) (int, int) {
	parts := strings.Split(rangeStr, ",")
	if len(parts) == 1 {
		n, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		return n, n
	}

	min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	if parts[1] == "" {
		return min, min + 5
	}
	max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return min, max
}

func (g *patternGenerator) randomPrintable() byte {
	printable := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return printable[g.rand.Intn(len(printable))]
}

func isQuantifier(b byte) bool {
	return b == '*' || b == '+' || b == '?' || b == '{'
}

// GenerateFromOneOf generates data from oneOf schema
func GenerateFromOneOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	if len(schemas) == 0 {
		return nil, fmt.Errorf("oneOf requires at least one schema")
	}

	idx := rand.Intn(len(schemas))
	return GenerateData(schemas[idx])
}

// GenerateFromAnyOf generates data from anyOf schema
func GenerateFromAnyOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	if len(schemas) == 0 {
		return nil, fmt.Errorf("anyOf requires at least one schema")
	}

	idx := rand.Intn(len(schemas))
	return GenerateData(schemas[idx])
}

// GenerateFromAllOf generates data merging all schemas in allOf
func GenerateFromAllOf(schemas []*openapi3.SchemaRef) (interface{}, error) {
	if len(schemas) == 0 {
		return nil, fmt.Errorf("allOf requires at least one schema")
	}

	result := make(map[string]interface{})

	for _, schemaRef := range schemas {
		if schemaRef.Value == nil {
			continue
		}

		data, err := GenerateData(schemaRef)
		if err != nil {
			return nil, fmt.Errorf("failed to generate allOf component: %w", err)
		}

		if obj, ok := data.(map[string]interface{}); ok {
			for k, v := range obj {
				result[k] = v
			}
		}
	}

	return result, nil
}

// GenerateAdvancedData generates data with advanced features
func GenerateAdvancedData(schema *openapi3.SchemaRef, fieldName string) (interface{}, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	s := schema.Value

	if s.Example != nil {
		return s.Example, nil
	}

	if len(s.OneOf) > 0 {
		return GenerateFromOneOf(s.OneOf)
	}

	if len(s.AnyOf) > 0 {
		return GenerateFromAnyOf(s.AnyOf)
	}

	if len(s.AllOf) > 0 {
		return GenerateFromAllOf(s.AllOf)
	}

	if len(s.Enum) > 0 {
		return s.Enum[rand.Intn(len(s.Enum))], nil
	}

	if s.Type == "string" && s.Pattern != "" {
		generated, err := GenerateFromPattern(s.Pattern)
		if err == nil {
			return generated, nil
		}
	}

	if s.Type == "string" && fieldName != "" {
		semanticType := DetectSemanticType(fieldName)
		if semanticType != SemanticUnknown {
			if value := GenerateBySemanticType(semanticType, s); value != nil {
				return value, nil
			}
		}
	}

	if s.Type == "object" {
		return generateAdvancedObject(s)
	}

	if s.Type == "array" {
		return generateAdvancedArray(s)
	}

	return GenerateData(schema)
}

func generateAdvancedObject(schema *openapi3.Schema) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	f := faker.New()

	for name, propSchema := range schema.Properties {
		data, err := GenerateAdvancedData(propSchema, name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate property %s: %w", name, err)
		}
		obj[name] = data
	}

	for _, allOfSchema := range schema.AllOf {
		if allOfSchema.Value == nil {
			continue
		}
		allOfObj, err := generateAdvancedObject(allOfSchema.Value)
		if err != nil {
			return nil, err
		}
		for k, v := range allOfObj {
			obj[k] = v
		}
	}

	if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		for i := 0; i < f.IntBetween(1, 3); i++ {
			key := f.Lorem().Word()
			val, err := GenerateAdvancedData(schema.AdditionalProperties.Schema, key)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
	}

	return obj, nil
}

func generateAdvancedArray(schema *openapi3.Schema) ([]interface{}, error) {
	f := faker.New()
	minItems := int(schema.MinItems)
	maxItems := 0
	if schema.MaxItems != nil {
		maxItems = int(*schema.MaxItems)
	}
	if maxItems == 0 {
		maxItems = minItems + 3
	}
	if minItems > maxItems {
		minItems = maxItems
	}
	if maxItems == 0 {
		maxItems = 3
	}

	count := f.IntBetween(minItems, maxItems)
	if count == 0 {
		count = 1
	}

	arr := make([]interface{}, count)
	for i := 0; i < count; i++ {
		item, err := GenerateAdvancedData(schema.Items, "")
		if err != nil {
			return nil, err
		}
		arr[i] = item
	}

	if schema.UniqueItems {
		arr = makeUnique(arr)
	}

	return arr, nil
}

func makeUnique(arr []interface{}) []interface{} {
	seen := make(map[string]bool)
	result := make([]interface{}, 0, len(arr))

	for _, item := range arr {
		key := fmt.Sprintf("%v", item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
