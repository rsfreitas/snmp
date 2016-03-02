# snmp
--
    import "github.com/PromonLogicalis/snmp"

Package snmp implements low-level support for SNMP with focus in SNMP agents.

At the encoding level it uses the PromonLogicalis/asn1 to parse and serialize
SNMP messages providing Go types for that.

The package also provides transport-independent support for creating custom SNMP
agents with small footprint.

A example of a simple SNMP UDP agent:

    package main

    import (
    	"errors"
    	"log"
    	"net"
    	"time"

    	"github.com/PromonLogicalis/asn1"
    	"github.com/PromonLogicalis/snmp"
    )

    func main() {
    	agent := snmp.NewAgent()

    	// Set the read-only and read-write communities
    	agent.SetCommunities("publ", "priv")

    	// Register a read-only OID.
    	since := time.Now()
    	agent.AddRoManagedObject(
    		// sysUpTime
    		asn1.Oid{1, 3, 6, 1, 2, 1, 1, 3, 0},
    		func(oid asn1.Oid) (interface{}, error) {
    			seconds := int(time.Now().Sub(since) / time.Second)
    			return seconds, nil
    		})

    	// Register a read-write OID.
    	name := "example"
    	agent.AddRwManagedObject(
    		// sysName
    		asn1.Oid{1, 3, 6, 1, 2, 1, 1, 5, 0},
    		func(oid asn1.Oid) (interface{}, error) {
    			return name, nil
    		},
    		func(oid asn1.Oid, value interface{}) error {
    			strValue, ok := value.(string)
    			if !ok {
    				return snmp.Errorf(snmp.BadValue, "invalid type")
    			}
    			name = strValue
    			return nil
    		})

    	// Bind to an UDP port
    	addr, err := net.ResolveUDPAddr("udp", ":161")
    	if err != nil {
    		log.Fatal(err)
    	}
    	conn, err := net.ListenUDP("udp", addr)
    	if err != nil {
    		log.Fatal(err)
    	}

    	// Serve requests
    	for {
    		buffer := make([]byte, 1024)
    		n, source, err := conn.ReadFrom(buffer)
    		if err != nil {
    			log.Fatal(err)
    		}

    		buffer, err = agent.ProcessDatagram(buffer[:n])
    		if err != nil {
    			log.Println(err)
    			continue
    		}

    		_, err = conn.WriteTo(buffer, source)
    		if err != nil {
    			log.Fatal(err)
    		}
    	}
    }

## Usage

```go
const (
	NoError             = 0
	TooBig              = 1
	NoSuchName          = 2
	BadValue            = 3
	ReadOnly            = 4
	GenErr              = 5
	NoAccess            = 6
	WrongType           = 7
	WrongLength         = 8
	WrongEncoding       = 9
	WrongValue          = 10
	NoCreation          = 11
	InconsistentValue   = 12
	ResourceUnavailable = 13
	CommitFailed        = 14
	UndoFailed          = 15
	AuthorizationError  = 16
	NotWritable         = 17
	InconsistentName    = 18
)
```
SNMP error codes.

#### func  Asn1Context

```go
func Asn1Context() *asn1.Context
```
Asn1Context returns a new allocated asn1.Context and registers all the choice
types necessary for SNMPv1 and SNMPv2.

#### type Agent

```go
type Agent struct {
}
```

Agent is a transport independent engine to process SNMP requests.

#### func  NewAgent

```go
func NewAgent() *Agent
```
NewAgent create and initialize an agent.

#### func (*Agent) AddRoManagedObject

```go
func (a *Agent) AddRoManagedObject(oid asn1.Oid, getter Getter) error
```
AddRoManagedObject registers a read-only managed object.

#### func (*Agent) AddRwManagedObject

```go
func (a *Agent) AddRwManagedObject(oid asn1.Oid, getter Getter,
	setter Setter) error
```
AddRwManagedObject registers a read-write managed object.

#### func (*Agent) ProcessDatagram

```go
func (a *Agent) ProcessDatagram(requestBytes []byte) (responseBytes []byte, err error)
```
ProcessRequest handles a binany SNMP message.

#### func (*Agent) ProcessMessage

```go
func (a *Agent) ProcessMessage(request *Message) (response *Message, err error)
```
ProcessMessage handles a SNMP Message.

#### func (*Agent) SetCommunities

```go
func (a *Agent) SetCommunities(public, private string)
```
SetCommunities defines the public and private communities.

#### func (*Agent) SetLogger

```go
func (a *Agent) SetLogger(logger *log.Logger)
```
SetLogger defines the logger used for internal messages.

#### type BulkPdu

```go
type BulkPdu struct {
	Id             int
	NonRepeaters   int
	MaxRepetitions int
	Variables      []Variable
}
```

BulkPdu is a generic type for other Protocol Data Units.

#### type Counter32

```go
type Counter32 uint32
```

Counter32 is a counter type.

#### type Counter64

```go
type Counter64 uint64
```

Counter64 is a counter type.

#### type EndOfMibView

```go
type EndOfMibView asn1.Null
```

EndOfMibView exception.

#### func (EndOfMibView) String

```go
func (e EndOfMibView) String() string
```

#### type Error

```go
type Error struct {
	Status  int
	Message string
}
```

Error is an error type that can be returned by a Getter or a Setter. When Error
is returned, it Status is used in the SNMP response.

#### func  Errorf

```go
func Errorf(status int, format string, values ...interface{}) Error
```
Errorf creates a new Error with a formatted message.

#### func (Error) Error

```go
func (e Error) Error() string
```

#### type GetBulkRequestPdu

```go
type GetBulkRequestPdu BulkPdu
```

GetBulkRequestPdu

#### type GetNextRequestPdu

```go
type GetNextRequestPdu Pdu
```

GetNextRequestPdu works similarly to GetRequestPdu, but it's returned the value
for the next valid Oid.

#### type GetRequestPdu

```go
type GetRequestPdu Pdu
```

GetRequestPdu is used to request data.

#### type GetResponsePdu

```go
type GetResponsePdu Pdu
```

GetResponsePdu is used in responses to SNMP requests.

#### type Getter

```go
type Getter func(oid asn1.Oid) (interface{}, error)
```

Getter is a function called to return a managed object value.

#### type InformRequestPdu

```go
type InformRequestPdu Pdu
```

InformRequestPdu

#### type IpAddress

```go
type IpAddress [4]byte
```

IpAddress is a IPv4 address.

#### func (IpAddress) String

```go
func (ip IpAddress) String() string
```
String returns a representation of IpAddress in dot notation.

#### type Message

```go
type Message struct {
	Version   int
	Community string
	Pdu       interface{} `asn1:"choice:pdu"`
}
```

Message is the top level element of the SNMP protocol.

#### type NoSuchInstance

```go
type NoSuchInstance asn1.Null
```

NoSuchInstance exception.

#### func (NoSuchInstance) String

```go
func (e NoSuchInstance) String() string
```

#### type NoSuchObject

```go
type NoSuchObject asn1.Null
```

NoSuchObject exception.

#### func (NoSuchObject) String

```go
func (e NoSuchObject) String() string
```

#### type Opaque

```go
type Opaque []byte
```

Opaque is a type for blobs.

#### type Pdu

```go
type Pdu struct {
	Id          int
	ErrorStatus int
	ErrorIndex  int
	Variables   []Variable
}
```

Pdu is a generic type for other Protocol Data Units.

#### type SetRequestPdu

```go
type SetRequestPdu Pdu
```

SetRequestPdu is used to request data to be updated.

#### type Setter

```go
type Setter func(oid asn1.Oid, value interface{}) error
```

Setter is a function called to set a managed object value.

#### type SnmpV1TrapPdu

```go
type SnmpV1TrapPdu struct {
	Enterprise   asn1.Oid
	AgentAddr    IpAddress
	GenericTrap  int
	SpecificTrap int
	Timestamp    TimeTicks
	Variables    []Variable
}
```

TrapPdu is used to register a trap in SNMPv1.

#### type SnmpV2TrapPdu

```go
type SnmpV2TrapPdu Pdu
```

SnmpV2TrapPdu is used to register a trap in SNMPv2.

#### type TimeTicks

```go
type TimeTicks uint32
```

TimeTicks is a type for time.

#### type Unsigned32

```go
type Unsigned32 uint32
```

Unsigned32 is an integer type.

#### type Variable

```go
type Variable struct {
	Name  asn1.Oid
	Value interface{} `asn1:"choice:val"`
}
```

Variable represents an entry of the variable bindings
