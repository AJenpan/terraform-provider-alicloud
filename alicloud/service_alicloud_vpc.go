package alicloud

import (
	"time"

	"fmt"

	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/terraform-providers/terraform-provider-alicloud/alicloud/connectivity"
)

type VpcService struct {
	client *connectivity.AliyunClient
}

func (s *VpcService) DescribeEipAddress(allocationId string) (eip vpc.EipAddress, err error) {

	args := vpc.CreateDescribeEipAddressesRequest()
	args.RegionId = string(s.client.Region)
	args.AllocationId = allocationId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeEipAddresses(args)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, allocationId, args.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		eips, _ := raw.(*vpc.DescribeEipAddressesResponse)
		if eips == nil || len(eips.EipAddresses.EipAddress) <= 0 {
			return WrapErrorf(Error(GetNotFoundMessage("EIP", allocationId)), NotFoundMsg, ProviderERROR)
		}
		eip = eips.EipAddresses.EipAddress[0]
		return nil
	})
	return
}

func (s *VpcService) DescribeEipAttachment(attachmentId string) (eip vpc.EipAddress, err error) {
	parts := strings.Split(attachmentId, ":")
	if len(parts) != 2 {
		err = WrapError(Error("invalid resource id"))
		return
	}
	eip, err = s.DescribeEipAddress(parts[0])
	if err != nil {
		err = WrapError(err)
		return
	}
	if eip.InstanceId != parts[1] {
		err = WrapErrorf(Error(GetNotFoundMessage("Eip Attachment", attachmentId)), NotFoundMsg, ProviderERROR)
	}

	return
}

func (s *VpcService) DescribeNatGateway(natGatewayId string) (nat vpc.NatGateway, err error) {

	args := vpc.CreateDescribeNatGatewaysRequest()
	args.RegionId = string(s.client.Region)
	args.NatGatewayId = natGatewayId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeNatGateways(args)
		})
		if err != nil {
			if IsExceptedError(err, InvalidNatGatewayIdNotFound) {
				return GetNotFoundErrorFromString(GetNotFoundMessage("Nat Gateway", natGatewayId))
			}
			return err
		}
		resp, _ := raw.(*vpc.DescribeNatGatewaysResponse)
		if resp == nil || len(resp.NatGateways.NatGateway) <= 0 {
			return GetNotFoundErrorFromString(GetNotFoundMessage("Nat Gateway", natGatewayId))
		}

		nat = resp.NatGateways.NatGateway[0]
		return nil
	})
	return
}

func (s *VpcService) DescribeVpc(vpcId string) (v vpc.DescribeVpcAttributeResponse, err error) {
	request := vpc.CreateDescribeVpcAttributeRequest()
	request.VpcId = vpcId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeVpcAttribute(request)
		})
		if err != nil {
			if IsExceptedErrors(err, []string{InvalidVpcIDNotFound, ForbiddenVpcNotFound}) {
				return WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
			}
			return WrapErrorf(err, DefaultErrorMsg, vpcId, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		resp, _ := raw.(*vpc.DescribeVpcAttributeResponse)
		if resp == nil || resp.VpcId != vpcId {
			return WrapErrorf(Error(GetNotFoundMessage("VPC", vpcId)), NotFoundMsg, ProviderERROR)
		}
		v = *resp
		return nil
	})
	return
}

func (s *VpcService) DescribeVswitch(vswitchId string) (v vpc.DescribeVSwitchAttributesResponse, err error) {
	request := vpc.CreateDescribeVSwitchAttributesRequest()
	request.VSwitchId = vswitchId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeVSwitchAttributes(request)
		})
		if err != nil {
			if IsExceptedError(err, InvalidVswitchIDNotFound) {
				return GetNotFoundErrorFromString(GetNotFoundMessage("VSwitch", vswitchId))
			}
			return err
		}
		resp, _ := raw.(*vpc.DescribeVSwitchAttributesResponse)
		if resp == nil || resp.VSwitchId != vswitchId {
			return GetNotFoundErrorFromString(GetNotFoundMessage("VSwitch", vswitchId))
		}
		v = *resp
		return nil
	})
	return
}

func (s *VpcService) DescribeSnatEntry(id string) (snat vpc.SnatTableEntry, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return snat, WrapError(err)
	}
	request := vpc.CreateDescribeSnatTableEntriesRequest()
	request.RegionId = string(s.client.Region)
	request.SnatTableId = parts[0]
	request.PageSize = requests.NewInteger(PageSizeLarge)

	for {
		invoker := NewInvoker()
		var response *vpc.DescribeSnatTableEntriesResponse
		err = invoker.Run(func() error {
			raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
				return vpcClient.DescribeSnatTableEntries(request)
			})
			response, _ = raw.(*vpc.DescribeSnatTableEntriesResponse)
			return err
		})

		//this special deal cause the DescribeSnatEntry can't find the records would be throw "cant find the snatTable error"
		//so judge the snatEntries length priority
		if err != nil {
			if IsExceptedErrors(err, []string{InvalidSnatTableIdNotFound, InvalidSnatEntryIdNotFound}) {
				return snat, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
			}
			return snat, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), response)

		if len(response.SnatTableEntries.SnatTableEntry) < 1 {
			break
		}

		for _, snat := range response.SnatTableEntries.SnatTableEntry {
			if snat.SnatEntryId == parts[1] {
				return snat, nil
			}
		}

		if len(response.SnatTableEntries.SnatTableEntry) < PageSizeLarge {
			break
		}
		if page, err := getNextpageNumber(request.PageNumber); err != nil {
			return snat, WrapError(err)
		} else {
			request.PageNumber = page
		}
	}

	return snat, WrapErrorf(Error(GetNotFoundMessage("SnatEntry", id)), NotFoundMsg, ProviderERROR)
}

func (s *VpcService) DescribeForwardEntry(forwardTableId string, forwardEntryId string) (entry vpc.ForwardTableEntry, err error) {

	args := vpc.CreateDescribeForwardTableEntriesRequest()
	args.RegionId = string(s.client.Region)
	args.ForwardTableId = forwardTableId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeForwardTableEntries(args)
		})
		//this special deal cause the DescribeSnatEntry can't find the records would be throw "cant find the snatTable error"
		//so judge the snatEntries length priority
		if err != nil {
			if IsExceptedErrors(err, []string{InvalidForwardEntryIdNotFound, InvalidForwardTableIdNotFound}) {
				return GetNotFoundErrorFromString(GetNotFoundMessage("Forward Entry", forwardTableId))
			}
			return err
		}
		resp, _ := raw.(*vpc.DescribeForwardTableEntriesResponse)
		if resp == nil || len(resp.ForwardTableEntries.ForwardTableEntry) <= 0 {
			return GetNotFoundErrorFromString(GetNotFoundMessage("Forward Entry", forwardTableId))
		}

		for _, forward := range resp.ForwardTableEntries.ForwardTableEntry {
			if forward.ForwardEntryId == forwardEntryId {
				entry = forward
				return nil
			}
		}

		return GetNotFoundErrorFromString(GetNotFoundMessage("Forward Entry", forwardTableId))
	})
	return
}

func (s *VpcService) QueryRouteTableById(routeTableId string) (rt vpc.RouteTable, err error) {
	request := vpc.CreateDescribeRouteTablesRequest()
	request.RouteTableId = routeTableId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeRouteTables(request)
		})
		if err != nil {
			return err
		}
		rts, _ := raw.(*vpc.DescribeRouteTablesResponse)
		if rts == nil || len(rts.RouteTables.RouteTable) == 0 ||
			rts.RouteTables.RouteTable[0].RouteTableId != routeTableId {
			return GetNotFoundErrorFromString(GetNotFoundMessage("Route Table", routeTableId))
		}

		rt = rts.RouteTables.RouteTable[0]
		return nil
	})
	return
}

func (s *VpcService) QueryRouteEntry(routeTableId, cidrBlock, nextHopType, nextHopId string) (rn vpc.RouteEntry, err error) {
	rt, err := s.QueryRouteTableById(routeTableId)
	if err != nil {
		return
	}

	for _, e := range rt.RouteEntrys.RouteEntry {
		if e.DestinationCidrBlock == cidrBlock && e.NextHopType == nextHopType && e.InstanceId == nextHopId {
			return e, nil
		}
	}
	return rn, GetNotFoundErrorFromString(GetNotFoundMessage("Route Entry", routeTableId))
}

func (s *VpcService) DescribeRouterInterface(regionId, interfaceId string) (ri vpc.RouterInterfaceType, err error) {
	request := vpc.CreateDescribeRouterInterfacesRequest()
	request.RegionId = regionId
	values := []string{interfaceId}
	filter := []vpc.DescribeRouterInterfacesFilter{{
		Key:   "RouterInterfaceId",
		Value: &values,
	},
	}
	request.Filter = &filter

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeRouterInterfaces(request)
		})
		if err != nil {
			return err
		}
		resp, _ := raw.(*vpc.DescribeRouterInterfacesResponse)
		if resp == nil || len(resp.RouterInterfaceSet.RouterInterfaceType) <= 0 ||
			resp.RouterInterfaceSet.RouterInterfaceType[0].RouterInterfaceId != interfaceId {
			return GetNotFoundErrorFromString(GetNotFoundMessage("Router Interface", interfaceId))
		}
		ri = resp.RouterInterfaceSet.RouterInterfaceType[0]
		return nil
	})
	return
}

func (s *VpcService) DescribeGrantRulesToCen(id string) (rule vpc.CbnGrantRule, err error) {
	request := vpc.CreateDescribeGrantRulesToCenRequest()
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return rule, WrapError(err)
	}
	cenId := parts[0]
	instanceId := parts[1]
	instanceType, err := GetCenChildInstanceType(instanceId)
	if err != nil {
		return rule, WrapError(err)
	}

	request.RegionId = s.client.RegionId
	request.InstanceId = instanceId
	request.InstanceType = instanceType

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeGrantRulesToCen(request)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		resp, _ := raw.(*vpc.DescribeGrantRulesToCenResponse)
		ruleList := resp.CenGrantRules.CbnGrantRule
		if resp == nil || len(ruleList) <= 0 {
			return WrapErrorf(Error(GetNotFoundMessage("GrantRules", id)), NotFoundMsg, ProviderERROR)
		}

		for ruleNum := 0; ruleNum <= len(resp.CenGrantRules.CbnGrantRule)-1; ruleNum++ {
			if ruleList[ruleNum].CenInstanceId == cenId {
				rule = ruleList[ruleNum]
				return nil
			}
		}

		return WrapErrorf(Error(GetNotFoundMessage("GrantRules", id)), NotFoundMsg, ProviderERROR)
	})
	return
}

func (s *VpcService) WaitForVpc(vpcId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	for {
		vpc, err := s.DescribeVpc(vpcId)
		if err != nil {
			return WrapError(err)
		}
		if vpc.Status == string(status) {
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return WrapError(Error(GetTimeoutMessage("VPC", string(status))))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForVSwitch(vswitchId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	for {
		vswitch, err := s.DescribeVswitch(vswitchId)
		if err != nil {
			return err
		}
		if vswitch.Status == string(status) {
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("VSwitch", string(status)))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForAllRouteEntries(routeTableId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	for {

		table, err := s.QueryRouteTableById(routeTableId)

		if err != nil {
			return err
		}

		success := true

		for _, routeEntry := range table.RouteEntrys.RouteEntry {
			if routeEntry.Status != string(status) {
				success = false
				break
			}
		}
		if success {
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("All Route Entries", string(status)))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForRouterInterface(regionId, interfaceId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	for {
		result, err := s.DescribeRouterInterface(regionId, interfaceId)
		if err != nil {
			if !NotFoundError(err) {
				return err
			}
		} else if result.Status == string(status) {
			break
		}

		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("Router Interface", string(status)))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForEip(allocationId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	for {
		eip, err := s.DescribeEipAddress(allocationId)
		if err != nil {
			if !NotFoundError(err) {
				return WrapError(err)
			}
		} else if eip.Status == string(status) {
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return WrapError(Error(GetTimeoutMessage("EIP", string(status))))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) DeactivateRouterInterface(interfaceId string) error {
	req := vpc.CreateDeactivateRouterInterfaceRequest()
	req.RouterInterfaceId = interfaceId
	_, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.DeactivateRouterInterface(req)
	})
	if err != nil {
		return fmt.Errorf("Deactivating RouterInterface %s got an error: %#v.", interfaceId, err)
	}
	return nil
}

func (s *VpcService) ActivateRouterInterface(interfaceId string) error {
	req := vpc.CreateActivateRouterInterfaceRequest()
	req.RouterInterfaceId = interfaceId
	_, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.ActivateRouterInterface(req)
	})
	if err != nil {
		return fmt.Errorf("Activating RouterInterface %s got an error: %#v.", interfaceId, err)
	}
	return nil
}

func (s *VpcService) WaitForForwardEntry(tableId, id string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	for {
		forward, err := s.DescribeForwardEntry(tableId, id)
		if err != nil {
			if !NotFoundError(err) {
				return WrapError(err)
			}
		} else if forward.Status == string(status) {
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return WrapError(Error(GetTimeoutMessage("Forward Entry", string(status))))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForSnatEntry(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		object, err := s.DescribeSnatEntry(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}

	}
}

// Flattens an array of vpc.public_ip_addresses into a []map[string]string
func (s *VpcService) FlattenPublicIpAddressesMappings(list []vpc.PublicIpAddresse) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		l := map[string]interface{}{
			"ip_address":    i.IpAddress,
			"allocation_id": i.AllocationId,
		}
		result = append(result, l)
	}

	return result
}
