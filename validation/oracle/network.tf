# Virtual Cloud Network (VCN)
resource "oci_core_vcn" "p1_vcn" {
  compartment_id = var.compartment_ocid
  cidr_block     = var.vcn_cidr
  display_name   = "p1-qualification-vcn"
  dns_label      = "p1vcn"

  tags = var.tags
}

# Internet Gateway
resource "oci_core_internet_gateway" "p1_igw" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.p1_vcn.id
  display_name   = "p1-igw"
  is_enabled     = true

  tags = var.tags
}

# Default route table - add route to IGW
resource "oci_core_default_route_table" "p1_default_rt" {
  manage_default_resource_id = oci_core_vcn.p1_vcn.default_route_table_id
  display_name               = "p1-default-rt"

  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.p1_igw.id
  }

  tags = var.tags
}

# Subnet
resource "oci_core_subnet" "p1_subnet" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.p1_vcn.id
  cidr_block     = var.subnet_cidr
  display_name   = "p1-qualification-subnet"
  dns_label      = "p1subnet"

  # Enable auto-assign public IP
  map_public_ip_on_launch = true

  route_table_id = oci_core_default_route_table.p1_default_rt.id

  tags = var.tags
}
