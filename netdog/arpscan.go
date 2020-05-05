package netdog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	log "github.com/sirupsen/logrus"
	"github.com/xkortex/vprint"
	"net"
	"sync"
	"time"
)

// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// arpscan implements ARP scanning of all interfaces' local networks using
// gopacket and its subpackages.  This example shows, among other things:
//   * Generating and sending packet data
//   * Reading in packet data and interpreting it
//   * Use of the 'pcap' subpackage for reading/writing

func ScanAllInterfaces(timeout float64, delay float64) (results interface{}, err error) {
	// Get a list of all interfaces.
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for _, iface := range ifaces {
		wg.Add(1)
		// Start up a Scan on each interface.
		go func(iface net.Interface) {
			defer wg.Done()
			if _, err := Scan(&iface, timeout, delay); err != nil {
				vprint.Printf("interface %v: %v", iface.Name, err)
			}
		}(iface)
	}
	// Wait for all interfaces' scans to complete.  They'll try to run
	// forever, but will stop on an error, so if we get past this Wait
	// it means all attempts to write have failed.
	wg.Wait()
	return "data (work in progress)", err

}

func ScanInterface(name string, timeout float64, delay float64) (results interface{}, err error) {
	// Get a list of all interfaces.
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	ifaceMap := map[string]net.Interface{}
	for _, iface := range ifaces {
		ifaceMap[iface.Name] = iface
	}
	var wg sync.WaitGroup

	if iface, ok := ifaceMap[name]; !ok {
		return nil, fmt.Errorf("Could not find interface: %s", name)
	} else {
		// Start up a Scan on each interface.
		wg.Add(1)

		go func(iface net.Interface) {
			vprint.Printf("Scanning:", iface.Name)
			defer wg.Done()

			if _, err := Scan(&iface, timeout, delay); err != nil {
				log.Errorf("interface %v: %v", iface.Name, err)
			}
		}(iface)
	}
	wg.Wait()
	return "data (work in progress)", err
}

// Scan scans an individual interface's local network for machines using ARP requests/replies.
//
// Scan loops forever, sending packets out regularly.  It returns an error if
// it's ever unable to write a packet.
func Scan(iface *net.Interface, timeout float64, delay float64) (arps []layers.ARP, err error) {
	// We just look for IPv4 addresses, so try to find if the interface has one.
	//ctx, cancel := context.WithTimeout(context.Background(), SecsToDuration(timeout))
	//defer cancel()

	var addr *net.IPNet
	if addrs, err := iface.Addrs(); err != nil {
		return nil, err
	} else {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					addr = &net.IPNet{
						IP:   ip4,
						Mask: ipnet.Mask[len(ipnet.Mask)-4:],
					}
					break
				}
			}
		}
	}
	// Sanity-check that the interface has a good address.
	if addr == nil {
		return nil, errors.New("no good IP network found")
	} else if addr.IP[0] == 127 {
		return nil, errors.New("skipping localhost")
	} else if addr.Mask[0] != 0xff || addr.Mask[1] != 0xff {
		return nil, errors.New("mask means network is too large")
	}
	vprint.Printf("Using network range %v for interface %v\n", addr, iface.Name)

	// Open up a pcap handle for packet reads/writes.
	handle, err := pcap.OpenLive(iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	// Start up a goroutine to read in packet data.
	stop := make(chan struct{})
	go readARP(handle, iface, stop)
	defer close(stop)
	// Write our Scan packets out to the handle.
	if err := writeARP(handle, iface, addr, delay); err != nil {
		log.Printf("error writing packets on %v: %v", iface.Name, err)
		return nil, err
	}
	// We don't know exactly how long it'll take for packets to be
	// sent back to us, but 10 seconds should be more than enough
	// time ;)
	time.Sleep(SecsToDuration(timeout))
	return nil, nil
}

func AggroScan(iface *net.Interface, timeout float64) (arps []layers.ARP, err error) {
	// We just look for IPv4 addresses, so try to find if the interface has one.
	//ctx, cancel := context.WithTimeout(context.Background(), SecsToDuration(timeout))
	//defer cancel()

	var addr *net.IPNet
	if addrs, err := iface.Addrs(); err != nil {
		return nil, err
	} else {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					addr = &net.IPNet{
						IP:   ip4,
						Mask: ipnet.Mask[len(ipnet.Mask)-4:],
					}
					break
				}
			}
		}
	}
	// Sanity-check that the interface has a good address.
	if addr == nil {
		return nil, errors.New("no good IP network found")
	} else if addr.IP[0] == 127 {
		return nil, errors.New("skipping localhost")
	} else if addr.Mask[0] != 0xff || addr.Mask[1] != 0xff {
		return nil, errors.New("mask means network is too large")
	}
	vprint.Printf("Using network range %v for interface %v\n", addr, iface.Name)

	// Open up a pcap handle for packet reads/writes.
	handle, err := pcap.OpenLive(iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	// Start up a goroutine to read in packet data.
	stop := make(chan struct{})
	go ReadInterface(handle, iface, stop)
	defer close(stop)
	// Write our Scan packets out to the handle.

	// We don't know exactly how long it'll take for packets to be
	// sent back to us, but 10 seconds should be more than enough
	// time ;)
	time.Sleep(SecsToDuration(timeout))
	return nil, nil
}

func ReadInterface(handle *pcap.Handle, iface *net.Interface, stop chan struct{}) {
	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	in := src.Packets()
	for {
		var packet gopacket.Packet
		select {
		case <-stop:
			return
		case packet = <-in:
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			if arp.Operation != layers.ARPReply || bytes.Equal([]byte(iface.HardwareAddr), arp.SourceHwAddress) {
				// This is a packet I sent.
				continue
			}
			// Note:  we might get some packets here that aren't responses to ones we've sent,
			// if for example someone else sends US an ARP request.  Doesn't much matter, though...
			// all information is good information :)
			//log.Printf("IP %v is at %v", net.IP(arp.SourceProtAddress), net.HardwareAddr(arp.SourceHwAddress))
			log.WithFields(log.Fields{
				"ip":  net.IP(arp.SourceProtAddress),
				"mac": fmt.Sprintf("%v", net.HardwareAddr(arp.SourceHwAddress)),
				"iface": iface.Name,
			}).Info()
		}
	}
}

// readARP watches a handle for incoming ARP responses we might care about, and prints them.
//
// readARP loops until 'stop' is closed.
func readARP(handle *pcap.Handle, iface *net.Interface, stop chan struct{}) {
	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	in := src.Packets()
	for {
		var packet gopacket.Packet
		select {
		case <-stop:
			return
		case packet = <-in:
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			if arp.Operation != layers.ARPReply || bytes.Equal([]byte(iface.HardwareAddr), arp.SourceHwAddress) {
				// This is a packet I sent.
				continue
			}
			// Note:  we might get some packets here that aren't responses to ones we've sent,
			// if for example someone else sends US an ARP request.  Doesn't much matter, though...
			// all information is good information :)
			//log.Printf("IP %v is at %v", net.IP(arp.SourceProtAddress), net.HardwareAddr(arp.SourceHwAddress))
			log.WithFields(log.Fields{
				"ip":  net.IP(arp.SourceProtAddress),
				"mac": fmt.Sprintf("%v", net.HardwareAddr(arp.SourceHwAddress)),
				"iface": iface.Name,
			}).Info()
		}
	}
}

// writeARP writes an ARP request for each address on our local network to the
// pcap handle.
func writeARP(handle *pcap.Handle, iface *net.Interface, addr *net.IPNet, delay float64) error {
	// Set up all the layers' fields we can.
	eth := layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(iface.HardwareAddr),
		SourceProtAddress: []byte(addr.IP),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
	}
	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	//fmt.Println(addr)
	for _, ip := range ips(addr) {
		arp.DstProtAddress = []byte(ip)
		gopacket.SerializeLayers(buf, opts, &eth, &arp)
		// Might want to trim to 42 because the camera does not respond to the padded one
		if err := handle.WritePacketData(buf.Bytes()); err != nil {
			return err
		}
		time.Sleep(SecsToDuration(delay))
	}
	return nil
}

// ips is a simple and not very good method for getting all IPv4 addresses from a
// net.IPNet.  It returns all IPs it can over the channel it sends back, closing
// the channel when done.
func ips(n *net.IPNet) (out []net.IP) {
	num := binary.BigEndian.Uint32([]byte(n.IP))
	mask := binary.BigEndian.Uint32([]byte(n.Mask))
	num &= mask
	//fmt.Printf("num: %x mask: %x", num, mask)
	for mask < 0xffffffff {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], num)
		out = append(out, net.IP(buf[:]))
		mask++
		num++
	}
	return
}
