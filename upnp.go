/*
Copyright (c) [2022] [Jaywang] [jaywang@petalmail.com]
[UPnPGO] is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:
         http://license.coscl.org.cn/MulanPSL2
THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
*/
package upnp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var defaultTransport http.RoundTripper = &http.Transport{
	Proxy: nil,
}

type UPNPProtocol string
type upnpFunc string

type UPNPService struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

type DeviceList struct {
	Device []Device `xml:"device"`
}

type ServiceList struct {
	Service []UPNPService `xml:"service"`
}

type Device struct {
	XMLName     xml.Name    `xml:"device"`
	DeviceType  string      `xml:"deviceType"`
	DeviceList  DeviceList  `xml:"deviceList"`
	ServiceList ServiceList `xml:"serviceList"`
}

type DevInfo struct {
	Device Device
}

type AddPPPReq struct {
	XMLName       xml.Name `xml:"s:Envelope"`
	Text          string   `xml:",chardata"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	S             string   `xml:"s,attr"`
	Body          struct {
		Text           string `xml:",chardata"`
		AddPortMapping struct {
			Text                      string `xml:",chardata"`
			U                         string `xml:"u,attr"`
			NewExternalPort           string `xml:"NewExternalPort"`
			NewInternalClient         string `xml:"NewInternalClient"`
			NewInternalPort           string `xml:"NewInternalPort"`
			NewProtocol               string `xml:"NewProtocol"`
			NewPortMappingDescription string `xml:"NewPortMappingDescription"`
			NewLeaseDuration          string `xml:"NewLeaseDuration"`
			NewEnabled                string `xml:"NewEnabled"`
		} `xml:"u:AddPortMapping"`
	} `xml:"s:Body"`
}

type PortMapReq struct {
	XMLName       xml.Name `xml:"s:Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                       string `xml:",chardata"`
		GetGenericPortMappingEntry struct {
			Text                string `xml:",chardata"`
			U                   string `xml:"u,attr"`
			NewPortMappingIndex string `xml:"NewPortMappingIndex"`
		} `xml:"u:GetGenericPortMappingEntry"`
	} `xml:"s:Body"`
}

type GetGenericPortMappingEntryResponse struct {
	Text                   string `xml:",chardata"`
	U                      string `xml:"u,attr"`
	RemoteHost             string `xml:"NewRemoteHost"`
	ExternalPort           string `xml:"NewExternalPort"`
	Protocol               string `xml:"NewProtocol"`
	InternalPort           string `xml:"NewInternalPort"`
	InternalClient         string `xml:"NewInternalClient"`
	Enabled                string `xml:"NewEnabled"`
	PortMappingDescription string `xml:"NewPortMappingDescription"`
	LeaseDuration          string `xml:"NewLeaseDuration"`
}

type PortMapResp struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                               string                             `xml:",chardata"`
		GetGenericPortMappingEntryResponse GetGenericPortMappingEntryResponse `xml:"GetGenericPortMappingEntryResponse"`
	} `xml:"Body"`
}

type DeletePortMapReq struct {
	XMLName       xml.Name `xml:"s:Envelope"`
	Text          string   `xml:",chardata"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	S             string   `xml:"s,attr"`
	Body          struct {
		Text              string `xml:",chardata"`
		DeletePortMapping struct {
			RemoteHost      string `xml:"NewRemoteHost"`
			Text            string `xml:",chardata"`
			U               string `xml:"u,attr"`
			NewExternalPort string `xml:"NewExternalPort"`
			NewInternalPort string `xml:"NewInternalPort"`
			NewProtocol     string `xml:"NewProtocol"`
		} `xml:"u:DeletePortMapping"`
	} `xml:"s:Body"`
}

type UPnP struct {
	loaclIP          net.IP
	wanPPPCURL       string
	wanPPPDomainName string
	appName          string
}

const (
	AddPortMapping             = upnpFunc("AddPortMapping")
	GetGenericPortMappingEntry = upnpFunc("GetGenericPortMappingEntry")
	DeletePortMapping          = upnpFunc("DeletePortMapping")
)

const (
	UDP = UPNPProtocol("UDP")
	TCP = UPNPProtocol("TCP")
)

func getTargetDev(dev *Device, target string) (*Device, error) {
	devList := dev.DeviceList.Device
	for i := 0; i < len(devList); i++ {
		if strings.Contains(devList[i].DeviceType, target) {
			return &devList[i], nil
		}
	}
	return &Device{}, errors.New("target not found")
}

func getTargetService(dev *Device, target string) (*UPNPService, error) {
	serList := dev.ServiceList.Service
	for i := 0; i < len(serList); i++ {
		if strings.Contains(serList[i].ServiceType, target) {
			return &serList[i], nil
		}
	}
	return &UPNPService{}, errors.New("target not found")
}

func (upnp *UPnP) soapPostReq(body []byte, upnpfunc upnpFunc) ([]byte, error) {
	req, err := http.NewRequest("POST", upnp.wanPPPCURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("")
	}
	req.Header.Set("Content-Type", "text/xml ; charset=\"utf-8\"")
	req.Header.Set("User-Agent", "VNET, UPnP/1.1, VNET/0.1")
	req.Header.Set("SOAPAction", "\"urn:"+upnp.wanPPPDomainName+":service:WANPPPConnection:1#"+string(upnpfunc)+"\"")
	req.Header.Set("Connection", "Close")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	cli := http.Client{
		Timeout:   time.Millisecond * 500,
		Transport: defaultTransport,
	}
	r, err := cli.Do(req)
	if err != nil {
		log.Println("UPNP Error: " + err.Error())
		return nil, errors.New("req error")
	}

	if r.StatusCode >= 400 {
		return nil, errors.New("Function " + string(upnpfunc) + " error. Code " + strconv.Itoa(r.StatusCode))
	}
	result, _ := ioutil.ReadAll(r.Body)
	return result, nil
}

func (upnp *UPnP) GetPortMapList() []GetGenericPortMappingEntryResponse {
	var portMapList []GetGenericPortMappingEntryResponse
	for i := 0; ; i++ {
		var portMapReq PortMapReq
		var portMapResp PortMapResp
		portMapReq.EncodingStyle = "http://schemas.xmlsoap.org/soap/encoding/"
		portMapReq.S = "http://schemas.xmlsoap.org/soap/envelope/"
		portMapReq.Body.GetGenericPortMappingEntry.U = "urn:" + upnp.wanPPPDomainName + ":service:WANPPPConnection:1"
		portMapReq.Body.GetGenericPortMappingEntry.NewPortMappingIndex = strconv.Itoa(i)
		body, _ := xml.Marshal(portMapReq)
		resp, err := upnp.soapPostReq(body, GetGenericPortMappingEntry)
		if err != nil {
			break
		}
		xml.Unmarshal(resp, &portMapResp)
		portMapList = append(portMapList, portMapResp.Body.GetGenericPortMappingEntryResponse)
	}
	return portMapList

}

func (upnp *UPnP) AddPort(loPort int, exPort int, protoc UPNPProtocol) {
	var addPPPReq AddPPPReq
	addPPPReq.EncodingStyle = "http://schemas.xmlsoap.org/soap/encoding/"
	addPPPReq.S = "http://schemas.xmlsoap.org/soap/envelope/"
	addPPPReq.Body.AddPortMapping.U = "urn:" + upnp.wanPPPDomainName + ":service:WANPPPConnection:1#AddPortMapping"
	addPPPReq.Body.AddPortMapping.NewInternalPort = strconv.Itoa(loPort)
	addPPPReq.Body.AddPortMapping.NewExternalPort = strconv.Itoa(exPort)
	addPPPReq.Body.AddPortMapping.NewPortMappingDescription = upnp.appName
	addPPPReq.Body.AddPortMapping.NewProtocol = string(protoc)
	addPPPReq.Body.AddPortMapping.NewLeaseDuration = "1000"
	addPPPReq.Body.AddPortMapping.NewEnabled = "1"
	addPPPReq.Body.AddPortMapping.NewInternalClient = upnp.loaclIP.To4().String()
	a, _ := xml.Marshal(addPPPReq)
	upnp.soapPostReq(a, AddPortMapping)
}

func (upnp *UPnP) DelPort(loPort int, exPort int, protoc UPNPProtocol) error {
	var deletePortMapReq DeletePortMapReq
	deletePortMapReq.EncodingStyle = "http://schemas.xmlsoap.org/soap/encoding/"
	deletePortMapReq.S = "http://schemas.xmlsoap.org/soap/envelope/"
	deletePortMapReq.Body.DeletePortMapping.U = "urn:" + upnp.wanPPPDomainName + ":service:WANPPPConnection:1#DeletePortMapping"
	deletePortMapReq.Body.DeletePortMapping.NewInternalPort = strconv.Itoa(loPort)
	deletePortMapReq.Body.DeletePortMapping.NewExternalPort = strconv.Itoa(exPort)
	deletePortMapReq.Body.DeletePortMapping.NewProtocol = string(protoc)
	a, _ := xml.Marshal(deletePortMapReq)
	_, err := upnp.soapPostReq(a, DeletePortMapping)
	return err
}

func NewUPnP(appName string) (UPnP, error) {
	var upnp UPnP
	upnp.appName = appName
	var ssdp = &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
	var addr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	var conn, _ = net.ListenUDP("udp", addr)
	defer conn.Close()
	var build strings.Builder
	build.WriteString("M-SEARCH * HTTP/1.1\r\n")
	build.WriteString("HOST: 239.255.255.250:1900\r\n")
	build.WriteString("ST: ssdp:all\r\n")
	build.WriteString("MAN: \"ssdp:discover\"\r\n")
	build.WriteString("MX: 2\r\n\r\n")
	message := []byte(build.String())
	for i := 0; i < 2; i++ {
		_, err := conn.WriteToUDP(message, ssdp)
		if err != nil {
			continue
		}
		var resp [4096]byte
		for {
			conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
			n, _, rerr := conn.ReadFromUDP(resp[:])
			if rerr != nil {
				return upnp, rerr
			}
			conn.SetDeadline(time.Now().Add(time.Minute))
			tag := "location:"
			res := strings.ToLower(string(resp[:n]))
			res = res[strings.Index(res, tag)+len(tag):]
			res = strings.Replace(res[:strings.Index(res, "\r\n")], " ", "", -1)
			upnp.wanPPPCURL, upnp.wanPPPDomainName, upnp.loaclIP, err = getDevInfo(res)
			if err == nil {
				// fmt.Println(upnp.wanPPPCURL, upnp.wanPPPDomainName, upnp.loaclIP.To4().String())
				return upnp, nil
			}
		}
	}
	return upnp, errors.New("error")
}

func getDevInfo(url string) (string, string, net.IP, error) {
	client := &http.Client{
		Transport: defaultTransport,
	}
	r, err := client.Get(url)
	var domainName string
	if err != nil {
		return url, domainName, nil, err
	}
	defer r.Body.Close()
	var devinfo DevInfo
	err = xml.NewDecoder(r.Body).Decode(&devinfo)
	if err != nil {
		return url, domainName, nil, err
	}

	root := &devinfo
	if !strings.Contains(root.Device.DeviceType, "InternetGatewayDevice:1") {
		return url, domainName, nil, errors.New("Not InternetGatewayDevice")
	}

	dev, err := getTargetDev(&root.Device, "WANDevice:1")
	if err != nil {
		return url, domainName, nil, err
	}

	subDev, err := getTargetDev(dev, "WANConnectionDevice:1")
	if err != nil {
		return url, domainName, nil, err
	}

	service, err := getTargetService(subDev, "WANPPPConnection:1")
	if err != nil {
		return url, domainName, nil, err
	}

	rootUrl := getUrlRoot(url)

	wanDevIP := net.ParseIP(strings.Split(rootUrl[strings.Index(rootUrl, "://")+3:], ":")[0])

	tt, err := net.Interfaces()
	if err != nil {
		return url, domainName, nil, err
	}
	for _, t := range tt {
		al, err := t.Addrs()
		if err != nil {
			continue
		}
		for _, a := range al {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.Contains(wanDevIP) {
				return rootUrl + service.ControlURL, strings.Split(service.ServiceType, ":")[1], ipnet.IP.To4(), nil
			}

		}

	}

	return "", "", nil, errors.New("error")
}

func getUrlRoot(url string) string {
	reg := regexp.MustCompile(`(\w+):\/\/([^/:]+)(:\d*)?`)
	result := reg.FindAllStringSubmatch(url, -1)
	return result[0][0]
}
