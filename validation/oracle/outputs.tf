output "vcn_id" {
  description = "VCN ID"
  value       = oci_core_vcn.p1_vcn.id
}

output "subnet_id" {
  description = "Subnet ID"
  value       = oci_core_subnet.p1_subnet.id
}

output "nsg_id" {
  description = "Network Security Group ID"
  value       = oci_core_network_security_group.p1_nsg.id
}

output "node_details" {
  description = "Details for all P1 nodes (IPs, IDs, access info)"
  value = {
    for node_name, instance in oci_core_instance.p1_nodes : node_name => {
      instance_id   = instance.id
      public_ip     = instance.public_ip
      private_ip    = instance.private_ip
      state         = instance.state
      shape         = instance.shape
      ad            = instance.availability_domain
      ssh_command   = "ssh -i <your-key> ubuntu@${instance.public_ip}"
    }
  }
}

output "p1_test_inventory" {
  description = "P1 test node inventory for provisioning scripts"
  value = {
    nodes = [
      for node_name, instance in oci_core_instance.p1_nodes : {
        name       = node_name
        public_ip  = instance.public_ip
        private_ip = instance.private_ip
        username   = "ubuntu"
      }
    ]
    vcn_cidr   = var.vcn_cidr
    subnet_cidr = var.subnet_cidr
    region      = var.region
  }

  depends_on = [oci_core_instance.p1_nodes]
}

output "p1_evidence_template" {
  description = "Template for P1 evidence collection domains"
  value = {
    for node_name, instance in oci_core_instance.p1_nodes : node_name => {
      machineDomain   = "distinct"
      vmDomain        = "distinct"
      operatorDomain  = "SAME/OCI"
      cloudDomain     = "SAME/OCI"
      regionDomain    = var.region
      adDomain        = var.availability_domain
      networkDomain   = "measured"
      powerDomain     = "UNKNOWN"
    }
  }
}
