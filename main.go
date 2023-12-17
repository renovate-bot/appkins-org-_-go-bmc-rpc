package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/paultyng/go-unifi/unifi"

	"github.com/appkins-org/go-bmc-rpc/config"
	"github.com/appkins-org/go-bmc-rpc/rpc"
)

var client *lazyClient = (*lazyClient)(nil)

var (
	port     int
	filePath string
	address  string
	cfg      config.Config
)

func main() {
	flag.IntVar(&port, "p", 5000, "port to listen on")
	flag.StringVar(&address, "a", "0.0.0.0", "address to listen on")
	flag.StringVar(&filePath, "c", "config.yaml", "configuration yaml file")
	flag.Parse()

	cfg, err := config.GetConfig(filePath)
	if err != nil {
		log.Fatalf("error reading YAML file: %v", err)
	}

	client = &lazyClient{
		user:     cfg.Username,
		pass:     cfg.Password,
		baseURL:  cfg.APIEndpoint,
		insecure: true,
	}

	r := mux.NewRouter()

	r.HandleFunc("/rpc", RPCHandler).Methods("POST")

	r.HandleFunc("/maaspower/{mac_address}/{port_idx}/query", QueryHandler).Methods("GET")

	http.Handle("/", r)

	fmt.Println("Server is running on http://0.0.0.0:5000")
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)

	if err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}

func getPort(ctx context.Context, macAddress string, portIdx string) (deviceId string, port unifi.DevicePortOverrides, err error) {
	deviceId = ""

	p, err := strconv.Atoi(portIdx)
	if err != nil {
		err = fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
		return
	}

	dev, err := client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		err = fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
		return
	}

	deviceId = dev.ID

	for _, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			port = pd
			break
		}
	}

	return
}

func setPortPower(ctx context.Context, macAddress string, portIdx string, state string) error {
	p, err := strconv.Atoi(portIdx)
	if err != nil {
		return fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
	}

	dev, err := client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		return fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
	}

	for i, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			switch state {
			case "on":
				if pd.PoeMode == "auto" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "auto"
				break
			case "off":
				if pd.PoeMode == "off" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "off"
				break
			}
		}
	}

	_, err = client.UpdateDevice(ctx, "default", dev)

	if err != nil {
		return fmt.Errorf("error updating device: %v", err)
	}

	return nil
}

func RPCHandler(w http.ResponseWriter, r *http.Request) {
	req := rpc.RequestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	machine := cfg.Machines[req.Host]

	rp := rpc.ResponsePayload{
		ID:   req.ID,
		Host: req.Host,
	}
	switch req.Method {
	case rpc.PowerGetMethod:
		state, err := GetPower(r.Context(), machine.MacAddress, machine.PortIdx)
		if err != nil {
			log.Fatalf("error getting power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			fmt.Fprintf(w, "error getting power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rp.Result = state
	case rpc.PowerSetMethod:
		p, ok := req.Params.(rpc.PowerSetParams)
		if !ok {
			log.Fatalf("error asserting params to PowerSetParams")
			fmt.Fprintf(w, "error asserting params to PowerSetParams")
			w.WriteHeader(http.StatusBadRequest)
		}
		state := p.State
		err := setPortPower(r.Context(), machine.MacAddress, machine.PortIdx, state)
		if err != nil {
			log.Fatalf("error setting power on for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			fmt.Fprintf(w, "error setting power on for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case rpc.BootDeviceMethod:
		p, ok := req.Params.(rpc.BootDeviceParams)
		if !ok {
			log.Fatalf("error asserting params to BootDeviceParams")
			fmt.Fprintf(w, "error asserting params to BootDeviceParams")
			w.WriteHeader(http.StatusBadRequest)
		}
		fmt.Fprintf(w, "boot device request for MAC Address %s, Port Index %s, Device %s, Persistent %t, EFIBoot %t", machine.MacAddress, machine.PortIdx, p.Device, p.Persistent, p.EFIBoot)

	case rpc.PingMethod:

		rp.Result = "pong"
	default:
		w.WriteHeader(http.StatusNotFound)
	}
	b, _ := json.Marshal(rp)
	w.Write(b)
}

func GetPower(ctx context.Context, macAddress string, portIdx string) (state string, err error) {
	_, port, err := getPort(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf("error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
		return
	}

	mode := port.PoeMode

	if mode == "auto" {
		state = "on"
	} else if mode == "off" {
		state = "off"
	}

	return
}

func QueryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macAddress := vars["mac_address"]
	portIdx := vars["port_idx"]

	_, port, err := getPort(r.Context(), macAddress, portIdx)
	if err != nil {
		fmt.Fprintf(w, "error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
		return
	}

	mode := port.PoeMode

	if mode == "auto" {
		fmt.Fprintf(w, "status : running")
		return
	} else if mode == "off" {
		fmt.Fprint(w, "status : stopped")
		return
	}

	fmt.Fprintf(w, "query request for MAC Address %s, Port Index %s", macAddress, portIdx)
}
