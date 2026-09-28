# Network Security Group
resource "oci_core_network_security_group" "p1_nsg" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.p1_vcn.id
  display_name   = "p1-qualification-nsg"

  tags = var.tags
}

# Ingress rule: SSH from anywhere (restrict in production)
resource "oci_core_network_security_group_security_rule" "p1_ssh_ingress" {
  network_security_group_id = oci_core_network_security_group.p1_nsg.id
  direction                 = "INGRESS"
  protocol                  = "6" # TCP
  description               = "SSH access"

  source      = "0.0.0.0/0"
  source_type = "CIDR_BLOCK"

  tcp_options {
    destination_port_range {
      min = 22
      max = 22
    }
  }
}

# Ingress rule: All traffic within subnet (inter-node communication)
resource "oci_core_network_security_group_security_rule" "p1_subnet_ingress" {
  network_security_group_id = oci_core_network_security_group.p1_nsg.id
  direction                 = "INGRESS"
  protocol                  = "all"
  description               = "Inter-node communication"

  source      = var.subnet_cidr
  source_type = "CIDR_BLOCK"
}

# Egress rule: All traffic outbound (allow internet access)
resource "oci_core_network_security_group_security_rule" "p1_egress" {
  network_security_group_id = oci_core_network_security_group.p1_nsg.id
  direction                 = "EGRESS"
  protocol                  = "all"
  description               = "Allow outbound traffic"

  destination      = "0.0.0.0/0"
  destination_type = "CIDR_BLOCK"
}
