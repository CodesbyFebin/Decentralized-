# Compute instances for P1 qualification (3-node Always Free topology)
resource "oci_core_instance" "p1_nodes" {
  for_each = var.nodes

  availability_domain = var.availability_domain
  compartment_id      = var.compartment_ocid
  display_name        = each.key

  shape = each.value.shape

  shape_config {
    ocpus         = each.value.ocpus
    memory_in_gbs = each.value.memory_gb
  }

  # Boot volume
  source_details {
    source_id   = data.oci_core_images.ubuntu_arm64.images[0].id
    source_type = "IMAGE"

    boot_volume_size_in_gbs = "60"

    kms_key_id = null
  }

  # Primary VNIC
  create_vnic_details {
    subnet_id        = oci_core_subnet.p1_subnet.id
    display_name     = "${each.key}-primary-vnic"
    private_ip       = each.value.private_ip
    public_ip        = "ORACLE_PROVIDED"
    assign_public_ip = true
    hostname_label   = each.key

    nsg_ids = [oci_core_network_security_group.p1_nsg.id]
  }

  # Metadata and cloud-init
  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data           = base64encode(file("${path.module}/cloud-init/dh-node.yaml"))
  }

  # Custom metadata for P1 test
  extended_metadata = {
    p1_node_name  = each.key
    p1_test_id    = "P1-ENDTOEND-A01"
    p1_region     = var.region
    p1_ad         = var.availability_domain
    p1_ocpus      = each.value.ocpus
    p1_memory_gb  = each.value.memory_gb
  }

  defined_tags = {
    ("${var.tags.project}") = {
      ("environment") = var.tags.environment
    }
  }

  freeform_tags = var.tags

  depends_on = [
    oci_core_internet_gateway.p1_igw,
    oci_core_network_security_group.p1_nsg
  ]
}

# Block storage volumes for additional node storage
resource "oci_core_volume" "p1_node_storage" {
  for_each = var.nodes

  availability_domain = var.availability_domain
  compartment_id      = var.compartment_ocid
  display_name        = "${each.key}-storage"
  size_in_gbs         = var.block_storage_size_gb

  tags = var.tags
}

# Attach block volumes to instances
resource "oci_core_volume_attachment" "p1_node_storage_attach" {
  for_each = var.nodes

  attachment_type = "paravirtualized"
  instance_id     = oci_core_instance.p1_nodes[each.key].id
  volume_id       = oci_core_volume.p1_node_storage[each.key].id
  device          = "/dev/oracleoci/oraclevdb"

  # Wait for volume to be created
  depends_on = [oci_core_volume.p1_node_storage]
}
