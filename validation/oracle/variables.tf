variable "tenancy_ocid" {
  description = "Oracle Cloud Tenancy OCID"
  type        = string
  sensitive   = true
}

variable "user_ocid" {
  description = "Oracle Cloud User OCID"
  type        = string
  sensitive   = true
}

variable "fingerprint" {
  description = "Public key fingerprint for OCI API access"
  type        = string
  sensitive   = true
}

variable "private_key" {
  description = "Private key for OCI API access (from API key)"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "OCI Region (e.g., us-phoenix-1, us-ashburn-1)"
  type        = string
  default     = "us-phoenix-1"
}

variable "compartment_ocid" {
  description = "Oracle Cloud Compartment OCID"
  type        = string
}

variable "availability_domain" {
  description = "Availability domain for instances"
  type        = string
  example     = "us-phoenix-1-AD-1"
}

variable "vcn_cidr" {
  description = "CIDR block for VCN"
  type        = string
  default     = "192.168.0.0/16"
}

variable "subnet_cidr" {
  description = "CIDR block for subnet"
  type        = string
  default     = "192.168.100.0/24"
}

variable "node_count" {
  description = "Number of compute nodes (typically 3 for P1 test)"
  type        = number
  default     = 3
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default = {
    environment = "p1-qualification"
    project     = "decentralized-host"
    managed_by  = "terraform"
  }
}

# Node-specific configurations
variable "nodes" {
  description = "P1 test node specifications (3-node Always Free topology)"
  type = map(object({
    ocpus     = number
    memory_gb = number
    shape     = string
    private_ip = string
  }))
  default = {
    dh-node-01 = {
      ocpus      = 1
      memory_gb  = 6
      shape      = "VM.Standard.A1.Flex"
      private_ip = "192.168.100.10"
    }
    dh-node-02 = {
      ocpus      = 1
      memory_gb  = 6
      shape      = "VM.Standard.A1.Flex"
      private_ip = "192.168.100.11"
    }
    dh-node-03 = {
      ocpus      = 2
      memory_gb  = 12
      shape      = "VM.Standard.A1.Flex"
      private_ip = "192.168.100.12"
    }
  }
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
  sensitive   = true
}

variable "block_storage_size_gb" {
  description = "Block storage size per node (GB)"
  type        = number
  default     = 50
}
