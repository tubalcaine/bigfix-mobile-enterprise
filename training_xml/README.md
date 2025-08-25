# Training XML Data (`training_xml`)

This directory contains real BigFix REST API XML responses collected from a BigFix lab environment. These XML files serve as training data for generating Go data structures and testing XML/JSON marshaling capabilities.

## Purpose

The training data supports the development of:
- **Go Struct Generation**: Create typed data structures from real API responses
- **XML/JSON Marshaling**: Ensure proper serialization/deserialization
- **API Coverage Testing**: Validate support for various BigFix API endpoints
- **Mobile App Development**: Enable browsing multiple BigFix servers from a single mobile application

## Files Overview

| File | Purpose | Description |
|------|---------|-------------|
| **bes.pl** | Perl script for BES API interaction | Legacy script for BigFix API testing and data collection |
| **besapi.pl** | Perl library for BigFix REST API | Reusable functions for API authentication and queries |
| **gen_get_script.py** | Python script generator | Creates shell scripts for systematic API data collection |
| **get_training_xml.sh** | Shell script for data collection | Automated script to fetch XML responses using curl |

## Data Collection Strategy

### 1. Plural Endpoints First
Start with "plural" endpoints that return lists of objects:
- `/api/computers` - All computers
- `/api/actions` - All actions  
- `/api/fixlets` - All fixlets
- `/api/tasks` - All tasks

### 2. Singular Endpoint Sampling
For each plural response, collect a representative sample of individual objects:
- `/api/computer/{id}` - Individual computer details
- `/api/action/{id}` - Individual action details
- `/api/fixlet/{site}/{id}` - Individual fixlet details

### 3. Complex Queries
Collect responses from complex relevance queries:
- Custom computer properties
- Action status queries  
- Site-specific content
- Multi-dimensional data

## Data Collection Tools

### Curl-Based Collection (`get_training_xml.sh`)

```bash
#!/bin/bash
# Example structure for systematic data collection

BIGFIX_SERVER="https://your-bigfix-server:52311"
USERNAME="console_user"
PASSWORD="your_password"

# Collect plural responses
curl -k -u "$USERNAME:$PASSWORD" "$BIGFIX_SERVER/api/computers" > computers.xml
curl -k -u "$USERNAME:$PASSWORD" "$BIGFIX_SERVER/api/actions" > actions.xml

# Collect singular samples
curl -k -u "$USERNAME:$PASSWORD" "$BIGFIX_SERVER/api/computer/1" > computer_1.xml
curl -k -u "$USERNAME:$PASSWORD" "$BIGFIX_SERVER/api/action/1" > action_1.xml
```

### Python Script Generator (`gen_get_script.py`)

Creates systematic collection scripts based on:
- Available API endpoints
- ID ranges for singular resources
- Query variations and parameters
- Output file naming conventions

### Perl Integration (`bes.pl`, `besapi.pl`)

Legacy tools for:
- BigFix session management
- Complex query construction
- Response validation
- Historical compatibility

## XML Structure Analysis

### Common Patterns

**Plural Responses:**
```xml
<BESAPI>
    <Computer Resource="https://...">
        <Property>Value</Property>
        <!-- Computer details -->
    </Computer>
    <Computer Resource="https://...">
        <!-- Additional computers -->
    </Computer>
</BESAPI>
```

**Singular Responses:**
```xml
<BESAPI>
    <Computer Resource="https://...">
        <Property>Detailed Value</Property>
        <SubElement>
            <NestedProperty>Value</NestedProperty>
        </SubElement>
        <!-- Extended details only in singular -->
    </Computer>
</BESAPI>
```

**Query Responses:**
```xml
<BESAPI>
    <Query Resource="https://...">
        <Result>
            <Answer>Query Result 1</Answer>
            <Answer>Query Result 2</Answer>
        </Result>
    </Query>
</BESAPI>
```

## Go Struct Generation

The collected XML data enables generation of Go structures like:

```go
type BESAPIResponse struct {
    XMLName   xml.Name    `xml:"BESAPI" json:"-"`
    Computers []Computer  `xml:"Computer" json:"computers,omitempty"`
    Actions   []Action    `xml:"Action" json:"actions,omitempty"`
    Query     *Query      `xml:"Query" json:"query,omitempty"`
}

type Computer struct {
    Resource   string      `xml:"Resource,attr" json:"resource"`
    ID         int64       `xml:"ID" json:"id"`
    Name       string      `xml:"Name" json:"name"`
    Properties []Property  `xml:"Property" json:"properties,omitempty"`
}
```

## Usage for Development

### 1. Collect Training Data
```bash
cd training_xml/
./get_training_xml.sh
```

### 2. Analyze XML Structures
```bash
# Find common patterns
grep -r "<Computer" *.xml | head -5
grep -r "Resource=" *.xml | head -5
```

### 3. Generate Go Structs
Use tools like `zek` or `xsdgen` with the collected XML:
```bash
zek computer_*.xml > computer_structs.go
```

### 4. Validate Marshaling
Test XML → Go → JSON conversion:
```go
var computer Computer
xml.Unmarshal(xmlData, &computer)
jsonData, _ := json.Marshal(computer)
```

## Quality Assurance

### Data Validation
- **Schema Compliance**: Verify against BigFix XSD schemas
- **Completeness**: Ensure all major API endpoints are covered
- **Variety**: Include different server configurations and data states
- **Edge Cases**: Collect error responses and unusual data patterns

### Testing Integration
- **Unit Tests**: Validate parsing of collected XML samples
- **Integration Tests**: Test end-to-end XML → Go → JSON → Mobile App flow
- **Performance Tests**: Measure parsing speed with real-world data sizes
- **Compatibility Tests**: Ensure support across BigFix versions

This training data provides the foundation for robust BigFix API integration and ensures the mobile application can handle real-world BigFix deployments effectively.
