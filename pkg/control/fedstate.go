package control

import (
	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
)

// Federation is the cluster's explicit, revocable trust with other
// independent Decentralized.Host installations. There is no global
// authority: every agreement is signed by the granting cluster's root and
// names the grantee root.
type Federation struct {
	// Granted: agreements this cluster signed, allowing a peer to place work here.
	Granted map[string]*FedAgreementRec `json:"granted"`
	// Held: agreements peers signed, allowing this cluster to place work there.
	Held map[string]*FedAgreementRec `json:"held"`
	// Inbound: placements peers made here (app name -> record).
	Inbound map[string]*FedPlacement `json:"inbound"`
	// Outbound: this cluster's apps placed on peers (app -> record).
	Outbound map[string]*FedOutbound `json:"outbound"`
}

// FedAgreement is signed by the granting cluster's root key (envelope kind
// "federation-agreement").
type FedAgreement struct {
	ID          string   `json:"id"`
	Grantor     string   `json:"grantor"`     // cluster name that runs the work
	GrantorRoot string   `json:"grantorRoot"` // its root key
	GrantorAPI  []string `json:"grantorApi"`  // where the grantee sends placements
	GrantorCA   string   `json:"grantorCa"`   // PEM root CA of the grantor; "" = its API is plain HTTP
	Grantee     string   `json:"grantee"`
	GranteeRoot string   `json:"granteeRoot"`
	Actions     []string `json:"actions"` // federation.place, federation.status, federation.withdraw
	Tiers       []string `json:"tiers"`   // workload classes the grantee may request
	Runtimes    []string `json:"runtimes"`
	MaxReplicas int64    `json:"maxReplicas"`
	MaxCPUMilli int64    `json:"maxCpuMilli"`
	MaxMemBytes int64    `json:"maxMemBytes"`
	NotBefore   int64    `json:"notBefore"`
	Expires     int64    `json:"expires"`
	Trust       string   `json:"trust"` // descriptive trust level
	Issued      int64    `json:"issued"`
}

type FedAgreementRec struct {
	Digest     string             `json:"digest"`
	Env        *envelope.Envelope `json:"env"`
	A          FedAgreement       `json:"a"`
	Revoked    bool               `json:"revoked"`
	RevokedAt  int64              `json:"revokedAt"`
	Revocation *envelope.Envelope `json:"revocation"`
	Received   int64              `json:"received"`
}

// FedPlacement is recorded on the granting side for each inbound app.
type FedPlacement struct {
	Peer       string             `json:"peer"`
	PeerRoot   string             `json:"peerRoot"`
	Agreement  string             `json:"agreement"`
	Request    string             `json:"request"` // digest of the signed placement request
	RequestEnv *envelope.Envelope `json:"requestEnv"`
	App        string             `json:"app"` // local app name (namespaced)
	RemoteApp  string             `json:"remoteApp"`
	Received   int64              `json:"received"`
	Withdrawn  bool               `json:"withdrawn"`
}

// FedOutbound is recorded on the requesting side.
type FedOutbound struct {
	App        string             `json:"app"`
	Peer       string             `json:"peer"`
	Agreement  string             `json:"agreement"`
	Replicas   int64              `json:"replicas"`
	Requested  int64              `json:"requested"`
	Accepted   bool               `json:"accepted"`
	Message    string             `json:"message"`
	LastStatus *envelope.Envelope `json:"lastStatus"`
	StatusAt   int64              `json:"statusAt"`
	Manifest   api.Manifest       `json:"manifest"`
}

func newFederation() Federation {
	return Federation{
		Granted: map[string]*FedAgreementRec{}, Held: map[string]*FedAgreementRec{},
		Inbound: map[string]*FedPlacement{}, Outbound: map[string]*FedOutbound{},
	}
}

func (f *Federation) ensure() {
	if f.Granted == nil {
		f.Granted = map[string]*FedAgreementRec{}
	}
	if f.Held == nil {
		f.Held = map[string]*FedAgreementRec{}
	}
	if f.Inbound == nil {
		f.Inbound = map[string]*FedPlacement{}
	}
	if f.Outbound == nil {
		f.Outbound = map[string]*FedOutbound{}
	}
}
