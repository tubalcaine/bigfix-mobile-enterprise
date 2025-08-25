# XML Schema Definitions (`xmlschema`)

This directory contains XML Schema Definition (XSD) files that define the structure and validation rules for BigFix XML data formats. These schemas are essential for generating Go data structures and ensuring XML parsing compatibility.

## Files Overview

| File | Purpose | Elements | Description |
|------|---------|----------|-------------|
| **BES.xsd** | BigFix Enterprise Suite schema | 984 lines | Defines core BigFix data structures for content, sites, and objects |
| **BESAPI.xsd** | BigFix REST API response schema | 1,270 lines | Defines REST API response formats for queries and data retrieval |

## Schema Details

### BES.xsd - Core BigFix Schema

**Primary Elements:**
- **Fixlet**: Security fixes and configuration policies
- **Task**: Administrative tasks and software deployment
- **Analysis**: Data collection and reporting queries
- **SingleAction**: Individual action execution definitions  
- **MultipleActionGroup**: Grouped action execution
- **Baseline**: Configuration baselines and compliance
- **ComputerGroup**: Computer targeting and grouping
- **Site**: Content organization and distribution
- **Property**: Custom property definitions

**Key Features:**
- Version 11.0.4.60 compatibility
- Qualified element and attribute formatting
- Support for complex nested data structures
- Extensible architecture for custom properties

### BESAPI.xsd - REST API Schema  

**Primary Elements:**
- **BESAPI**: Root element for all API responses
- **Computer**: Computer information and properties
- **Action**: Action status and execution details
- **Query**: Relevance query results and answers
- **Fixlet**: Content metadata and availability
- **ReplicationServer**: Server topology information
- **WebReports**: Reporting server configuration
- **ManualComputerGroup**: Computer group management

**Key Features:**
- Resource attribute linking for REST endpoints
- Timestamp and modification tracking
- Nested result structures for complex queries
- Support for plural and singular response formats

## Code Generation Usage

### Go Struct Generation

These schemas enable automatic Go struct generation using tools like `xgen`:

```bash
# Generate Go structs from BES schema
xgen -i BES.xsd -o bes_structs.go -p main -l Go

# Generate Go structs from BESAPI schema  
xgen -i BESAPI.xsd -o besapi_structs.go -p main -l Go
```

### Integration with Training Data

The schemas work in conjunction with training XML data to ensure:
- **Type Safety**: Generated structs match real XML responses
- **Validation**: Schema compliance checking during parsing
- **Coverage**: Support for all BigFix API response formats
- **Evolution**: Schema versioning tracks BigFix platform updates

## Schema Structure Analysis

### Common Patterns

**Resource Attribution:**
```xml
<xs:attribute name="Resource" type="xs:normalizedString"/>
```
- Links XML elements to REST API endpoints
- Enables navigation between related objects
- Provides unique identification for caching

**Nested Data Structures:**
```xml
<xs:complexType>
  <xs:sequence>
    <xs:element name="Property" maxOccurs="unbounded"/>
  </xs:sequence>
</xs:complexType>
```
- Hierarchical data organization
- Support for variable-length collections
- Flexible property extensions

**Timestamp Handling:**
```xml
<xs:attribute name="LastModified" type="xs:normalizedString"/>
```
- Modification tracking for cache invalidation
- Synchronization support across servers
- Version control for content updates

## Development Integration

### XML Parsing Validation

```go
// Validate XML against schema before parsing
func ValidateXML(xmlData []byte, schemaFile string) error {
    // Schema validation logic
    return nil
}

// Parse validated XML into Go structs
func ParseBESAPI(xmlData []byte) (*BESAPIResponse, error) {
    var response BESAPIResponse
    err := xml.Unmarshal(xmlData, &response)
    return &response, err
}
```

### Type Generation Pipeline

1. **Schema Analysis**: Parse XSD files to understand structure
2. **Struct Generation**: Create Go types with proper XML tags
3. **Validation**: Test against training XML data
4. **Integration**: Include in pkg/bfrest parsing logic

## Version Compatibility

**Current Schema Version**: 11.0.4.60
- Compatible with BigFix 11.x platforms
- Backward compatibility with 10.x responses
- Forward compatibility considerations for 12.x

**Schema Evolution:**
- New elements added as optional to maintain compatibility
- Deprecated elements marked but not removed
- Version tracking in schema namespace

## Usage in BEM Server

The schemas support the BEM server's XML processing capabilities:

**Response Processing:**
- Validate incoming BigFix XML responses
- Parse into strongly-typed Go structures
- Convert to JSON for mobile client consumption
- Cache with proper type information

**Error Handling:**
- Schema validation errors for malformed responses
- Type conversion errors for unexpected data
- Fallback parsing for schema evolution

This schema foundation ensures robust, type-safe XML processing throughout the BigFix Enterprise Mobile ecosystem.