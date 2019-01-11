package tfit

const EC2_ROUTE_TABLE = `{{ if . }}
  {{- range .}}
resource "aws_route_table" "{{ .Id }}" {
  vpc_id = "{{ .VpcId }}"

  {{- if .Tags }}
  tags {
    {{- range .Tags }}
    "{{ .Key }}" = "{{ .Value }}"
    {{- end }}
  }
  {{- end }}

  {{- if .PropagatingVgws }}
  propagating_vgws = [{{ .PropagatingVgws | makeTerraformList }}]
  {{- end }}

  {{- if .Routes }}
    {{- range .Routes }}
  route {
    {{- if .CIDRBlock }}
    cidr_block = "{{ .CIDRBlock }}"
    {{- end }}
    {{- if .IPv6CIDRBlock }}
    ipv6_cidr_block = "{{ .IPv6CIDRBlock }}"
    {{- end }}
    {{- if .VpcPeeringConnectionId }}
    vpc_peering_connection_id = "{{ .VpcPeeringConnectionId }}"
    {{- end }}
    {{- if .TransitGatewayId }}
    transit_gateway_id = "{{ .TransitGatewayId }}"
    {{- end }}
    {{- if .NetworkInterfaceId }}
    network_interface_id = "{{ .NetworkInterfaceId }}"
    {{- end }}
    {{- if .NatGatewayId }}
    nat_gateway_id = "{{ .NatGatewayId }}"
    {{- end }}
    {{- if .InstanceId }}
    instance_id = "{{ .InstanceId }}"
    {{- end }}
    {{- if .GatewayId }}
    gateway_id = "{{ .GatewayId }}"
    {{- end }}
    {{- if .EgressOnlyInternetGatewayId }}
    egress_only_gateway_id = "{{ .EgressOnlyInternetGatewayId }}"
    {{- end }}
  }
    {{- end }}
  {{- end }}

}
  {{- end }}
{{ end }}`
