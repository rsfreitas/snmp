// TODO Documentation
// TODO Support for traps
// TODO More flexible ACL and authentication mechanism.
// TODO Use the origin to process ACLs and authentication.
// TODO Support for SNMPv2.
package snmp

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"sort"

	"github.com/PromonLogicalis/asn1"
)

// Getter is a function called to return a managed object value.
type Getter func(oid asn1.Oid) (interface{}, error)

// Setter is a function called to set a managed object value.
type Setter func(oid asn1.Oid, value interface{}) error

// Agent is a transport independent engine to process SNMP requests.
type Agent struct {
	log      *log.Logger
	ctx      *asn1.Context
	handlers []managedObject
	public   string
	private  string
}

// NewAgent create and initialize an agent.
func NewAgent() *Agent {
	a := &Agent{ctx: Asn1Context()}
	a.SetLogger(nil)
	a.SetCommunities("public", "private")
	return a
}

// SetLogger
func (a *Agent) SetLogger(logger *log.Logger) {
	if logger == nil {
		logger = log.New(ioutil.Discard, "", 0)
	}
	a.log = logger
	a.ctx.SetLogger(logger)
}

// SetCommunities defines the public and private communities.
func (a *Agent) SetCommunities(public, private string) {
	a.public, a.private = public, private
}

// checkCommunity handles "authentication" and acls
func (a *Agent) checkCommunity(community string) (rw bool, err error) {

	// Access check. Right now only read-only community is implemented
	if community != a.public && community != a.private {
		// The agent should ignore invalid communities
		err = fmt.Errorf("invalid community \"%s\"", community)
		return
	}

	// Super complex ACLs
	if community == a.private {
		rw = true
	}
	return
}

// AddRoManagedObject registers a read-only managed object.
func (a *Agent) AddRoManagedObject(oid asn1.Oid, getter Getter) error {
	return a.AddRwManagedObject(oid, getter, nil)
}

// AddRwManagedObject registers a read-write managed object.
func (a *Agent) AddRwManagedObject(oid asn1.Oid, getter Getter,
	setter Setter) error {

	if getter == nil {
		return fmt.Errorf("A managed object should have at least a getter.")
	}
	if setter == nil {
		setter = func(oid asn1.Oid, value interface{}) error {
			return Errorf(NotWritable, "OID %s is not writable", oid)
		}
	}
	if a.getManagedObject(oid, false) != nil {
		return fmt.Errorf("OID %d is already registered.", oid)
	}
	h := managedObject{oid, nil, getter, setter}
	a.handlers = append(a.handlers, h)
	sort.Sort(sortableManagedObjects(a.handlers))
	return nil
}

// managedObject represents a registered managed object.
type managedObject struct {
	oid asn1.Oid
	// TODO Add type check inside the agent processing.
	typ reflect.Type
	get Getter
	set Setter
}

// sortableManagedObjects is a helper type to sort managed objects slices.
type sortableManagedObjects []managedObject

func (h sortableManagedObjects) Len() int      { return len(h) }
func (h sortableManagedObjects) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h sortableManagedObjects) Less(i, j int) bool {
	return h[i].oid.Cmp(h[j].oid) < 0
}

// getManagedObject returns the exact managed object for the given OID when
// next=false  or the next object when next=true.
func (a *Agent) getManagedObject(oid asn1.Oid, next bool) *managedObject {
	for _, h := range a.handlers {
		cmp := oid.Cmp(h.oid)
		if (!next && cmp == 0) || (next && cmp < 0) {
			return &h
		}
		if !next && cmp < 0 {
			break
		}
	}
	return nil
}

// ProcessRequest handles a binany SNMP message.
func (a *Agent) ProcessDatagram(requestBytes []byte) (responseBytes []byte, err error) {
	// Decode message. Invalid messages are discarded
	ctx := Asn1Context()
	msg, err := decodeMessage(ctx, requestBytes)
	if err != nil {
		return
	}

	// SNMPv1 for now
	if msg.Version != 0 && msg.Version != 1 {
		// Discard SNMPv2 messages
		err = fmt.Errorf("invalid SNMP version %d", msg.Version)
		return
	}

	rw, err := a.checkCommunity(msg.Community)
	if err != nil {
		return
	}

	// Dispatch each type of PDU
	a.log.Printf("request: %#v\n", msg)
	var res GetResponsePdu
	switch pdu := msg.Pdu.(type) {
	case GetRequestPdu:
		res = a.processPdu(Pdu(pdu), false, false)
	case GetNextRequestPdu:
		res = a.processPdu(Pdu(pdu), true, false)
	case SetRequestPdu:
		if rw {
			res = a.processPdu(Pdu(pdu), false, true)
		} else {
			res = GetResponsePdu(pdu)
			res.ErrorIndex = 1
			res.ErrorStatus = NoSuchName
		}
	default:
		// SNMPv2 PDUs are ignored
		err = fmt.Errorf("PDU not supported: %T", msg.Pdu)
		return
	}

	msg.Pdu = res
	a.log.Printf("response: %#v\n", msg)
	responseBytes, err = encodeMessage(ctx, msg)
	return
}

// processPdu handles SNMPv1 requests with exception of SnmpV1TrapPdu.
func (a *Agent) processPdu(pdu Pdu, next bool, set bool) GetResponsePdu {

	// Keep returned values in a separated slice for a Get request
	var variables []Variable

	var err error
	res := GetResponsePdu(pdu)
	for i, v := range pdu.Variables {
		a.log.Printf("oid: %s\n", v.Name)
		// Retrieve the managed object
		h := a.getManagedObject(v.Name, next)
		if h == nil {
			res.ErrorIndex = i + 1
			res.ErrorStatus = NoSuchName
			return res
		}
		// Set or get the value
		var value interface{}
		if set {
			err = h.set(h.oid, v.Value)
		} else {
			value, err = h.get(h.oid)
		}
		if err != nil {
			res.ErrorIndex = i + 1
			if e, ok := err.(Error); ok {
				res.ErrorStatus = e.Status
			} else {
				res.ErrorStatus = GenErr
			}
			return res
		}
		// Values returned by a Get are kept in a separated list. If an error
		// occurs the original list of variables should be returned.
		if !set {
			variables = append(variables, Variable{h.oid, value})
		}
	}
	if !set {
		// Update all values, since all variables were processed without error:
		res.Variables = variables
	}
	return res
}

// Error is an error type that can be returned by a Getter or a Setter. When
// Error is returned, it Status is used in the SNMP response.
type Error struct {
	Status  int
	Message string
}

var _ error = Error{}

func (e Error) Error() string {
	return fmt.Sprintf("%s (status: %d)", e.Message, e.Status)
}

// Errorf creates a new Error with a formatted message.
func Errorf(status int, format string, values ...interface{}) Error {
	return Error{
		Status:  status,
		Message: fmt.Sprintf(format, values...),
	}
}
