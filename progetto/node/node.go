package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

var N int

type Node struct {
	ID            int
	Predecessor   string
	Successor     string
	Resources     map[int](string)
	Ip            string
	fingerTable   []int
	fingerTableIP map[int]string // Inizializzazione della mappa per gli IP
}

type Arg struct {
	ID int
	Ip string
}

var curr_node Node
var stopChan = make(chan struct{})

type RegisterNode string

var serverAddr = "s_registry:8000"

//var serverAddr = "0.0.0.0:8000"

// Print the finger table for a node.
func printFingerTable(node *Node) {

	fmt.Println()
	fmt.Printf("Finger Table of %d:", node.ID)
	fmt.Println()
	for i := 1; i < len(node.fingerTable); i++ {
		fmt.Printf("[%d | %d]", i, node.fingerTable[i])
		fmt.Println()
	}
}

func Hash(input string) int {
	hash := sha256.Sum256([]byte(input))
	hashValue := 0
	for _, b := range hash {
		hashValue = (hashValue << 8) | int(b)
	}

	// Assicurati che il risultato sia non negativo
	if hashValue < 0 {
		hashValue = -hashValue
	}

	return hashValue % N
}

/*func Hash(input string) int {
	hash := sha256.Sum256([]byte(input))
	return int(hash[0]) % N
}*/

func NewNode(ip_add string, num_bit int) Node {

	//registro il nodo e lo inizializz.
	client, err := rpc.DialHTTP("tcp", serverAddr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var reply *Node
	N = num_bit
	//N = int(math.Pow(2, float64(num_bit)))
	curr_node.Ip = ip_add
	curr_node.ID = Hash(ip_add)

	err = client.Call("Manager.RegisterNode", curr_node, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}

	curr_node.Resources = make(map[int](string))
	curr_node.Successor = reply.Successor
	curr_node.Predecessor = reply.Predecessor

	return curr_node
}

func shouldReturnSuccessor(distance int) bool {
	return (distance <= Hash(curr_node.Successor) && distance > curr_node.ID) ||
		(distance > curr_node.ID && curr_node.ID > Hash(curr_node.Successor)) ||
		(curr_node.ID > Hash(curr_node.Successor) && distance < curr_node.ID && distance <= Hash(curr_node.Successor))
}

func calculateDistance(nodeID, index, num_bit int) int {
	result := nodeID + (1 << (index - 1))
	modValue := 1 << num_bit
	return result % modValue
}
func shouldUseSuccessor(nodeID, successorHash, distance int) bool {
	return (distance > nodeID && distance <= successorHash) ||
		(distance > nodeID && nodeID > successorHash) ||
		(nodeID > successorHash && distance < nodeID && distance <= successorHash)
}
func (t *RegisterNode) GetSuccessor(distance int, reply *Arg) error {

	if shouldReturnSuccessor(distance) {
		var arg Arg
		arg.ID = Hash(curr_node.Successor)
		arg.Ip = curr_node.Successor
		*reply = arg

		//*reply = Hash(curr_node.Successor)
	} else {
		// Inoltra la richiesta al successore del successore
		client, err := rpc.DialHTTP("tcp", curr_node.Successor)
		if err != nil {
			log.Fatal("Errore nella connessione del client: ", err)
		}
		err = client.Call("RegisterNode.GetSuccessor", distance, &reply)
		if err != nil {
			log.Fatal("Errore nell'invocazione del client: ", err)
		}
		client.Close()
	}
	return nil
}

func getFingerTableEntry(node *Node, distance int) (*Arg, error) {
	var reply *Arg

	client, err := rpc.DialHTTP("tcp", node.Successor)
	if err != nil {
		log.Fatal("Errore nella connessione del client: ", err)
	}

	err = client.Call("RegisterNode.GetSuccessor", distance, &reply)
	if err != nil {
		log.Fatal("Errore nell'invocazione del client: ", err)
	}

	client.Close()

	return reply, nil
}

// nella FingerTable salvo l'id e l'ip ma poi devo cambiare tutto
func InitializeFingerTable(node *Node, num_bit int) error {

	var arg *Arg

	num_entry := num_bit + 1
	fingerTable := make([]int, num_entry)
	fingerTableIp := make(map[int]string)

	fingerTable[0] = node.ID
	fingerTable[1] = Hash(node.Successor)

	fingerTableIp[Hash(node.Successor)] = node.Successor

	for i := 2; i < num_entry; i++ {

		distance := calculateDistance(node.ID, i, num_bit)

		if node.ID == Hash(node.Successor) {
			fingerTable[i] = node.ID
		} else if shouldUseSuccessor(node.ID, Hash(node.Successor), distance) {
			fingerTable[i] = Hash(node.Successor)
		} else {
			arg, _ = getFingerTableEntry(node, distance)
			if arg != nil {
				fingerTable[i] = arg.ID
				if _, exists := fingerTableIp[arg.ID]; !exists {
					// La chiave non esiste nella mappa, quindi possiamo aggiungerla
					fingerTableIp[arg.ID] = arg.Ip
				}
			}
		}
	}

	curr_node.fingerTable = fingerTable[1:]
	curr_node.fingerTableIP = fingerTableIp
	//printFingerTable(node)
	fmt.Println(curr_node)

	return nil

}

func (t *RegisterNode) UpdatePred(node Node, reply *map[int](string)) error {

	curr_node.Predecessor = node.Ip
	fmt.Println()

	for key, value := range curr_node.Resources {
		if !MyKey(key) {

			(*reply)[key] = value
			delete(curr_node.Resources, key)
		}
	}

	fmt.Println(curr_node)
	return nil
}

func (t *RegisterNode) UpdateSucc(node Node, reply *map[int](string)) error {

	curr_node.Successor = node.Ip

	fmt.Println()
	fmt.Println(curr_node)
	return nil
}

// verifico se una chiave appartiene al mio range di competenza
func MyKey(key int) bool {

	if curr_node.ID == Hash(curr_node.Predecessor) {
		return true
	}
	if curr_node.ID <= Hash(curr_node.Predecessor) {
		if key > Hash(curr_node.Predecessor) || key <= curr_node.ID {
			return true
		}
		return false
	}
	if key > Hash(curr_node.Predecessor) && key <= curr_node.ID {
		return true
	}
	return false

}

// recupero una risorsa
func (t *RegisterNode) GetData(key int, reply *string) error {

	fmt.Println()
	fmt.Printf("Mi hanno contattato per la chiave %d ", key)
	fmt.Println()

	if MyKey(key) {

		data, exists := curr_node.Resources[key]
		if !exists {
			data = ""
		}
		*reply = data
		delete(curr_node.Resources, key)

	} else if (key > curr_node.ID) && (key <= Hash(curr_node.Successor)) || ((key > curr_node.ID) || (key <= Hash(curr_node.Successor))) {
		s_Ip := curr_node.Successor
		client, err := rpc.DialHTTP("tcp", s_Ip)
		if err != nil {
			log.Fatal("Client connection error: ", err)
		}

		err = client.Call("RegisterNode.GetData", key, &reply)
		if err != nil {
			log.Fatal("Client invocation error: ", err)
		}

		//ALTRIMENTI CONSULTO LA FINGERTABLE
	} else {

		for i := 1; i < len(curr_node.fingerTable); i++ {

			if key >= curr_node.fingerTable[i-1] && key < curr_node.fingerTable[i] {

				//devo prendere l'ip di curr_node.fingerTable[i]
				s_ID := curr_node.fingerTable[i-1]
				s_Ip := curr_node.fingerTableIP[s_ID]

				client, err := rpc.DialHTTP("tcp", s_Ip)
				if err != nil {
					log.Fatal("Client connection error: ", err)
				}

				err = client.Call("RegisterNode.GetData", key, &reply)
				if err != nil {
					log.Fatal("Client invocation error: ", err)
				}

			} else {
				continue
			}
		}
	}

	//fmt.Println(curr_node.Resources)
	return nil
}

// aggiungo una stringa
func (t *RegisterNode) AddData(data string, reply *string) error {

	key := Hash(data)
	fmt.Println()
	fmt.Printf("Mi hanno contattato per la stringa %s con la chiave %d", data, key)
	fmt.Println()

	if MyKey(key) {

		_, exists := curr_node.Resources[key]
		if exists {
			*reply = "Esiste già una stringa con la stessa chiave."

		} else {
			curr_node.Resources[key] = data
			*reply = "La stringa inserita è stata memorizzata con successo. La sua chiave è: " + strconv.Itoa(key)
		}
		// CONTROLLO SE APPARTIENE AL SUCCESSORE
		//il successore deve controllare se non esiste già
	} else if ((key > curr_node.ID) && (key <= Hash(curr_node.Successor))) || ((key > curr_node.ID) || (key <= Hash(curr_node.Successor))) {
		s_Ip := curr_node.Successor
		client, err := rpc.DialHTTP("tcp", s_Ip)
		if err != nil {
			log.Fatal("Client connection error: ", err)
		}

		err = client.Call("RegisterNode.AddData", data, &reply)
		if err != nil {
			log.Fatal("Client invocation error: ", err)
		}
		//*reply = "La stringa inserita è stata memorizzata con successo. La sua chiave è: " + strconv.Itoa(key)
		//ALTRIMENTI CONSULTO LA FINGERTABLE
	} else {
		//fmt.Println("Sto nel else:", len(curr_node.fingerTable))

		for i := 1; i < len(curr_node.fingerTable); i++ {

			if key >= curr_node.fingerTable[i-1] && key < curr_node.fingerTable[i] {

				//devo prendere l'ip di curr_node.fingerTable[i]
				s_ID := curr_node.fingerTable[i-1]
				s_Ip := curr_node.fingerTableIP[s_ID]

				client, err := rpc.DialHTTP("tcp", s_Ip)
				if err != nil {
					log.Fatal("Client connection error: ", err)
				}

				err = client.Call("RegisterNode.AddData", data, &reply)
				if err != nil {
					log.Fatal("Client invocation error: ", err)
				}

			} else {
				continue
			}
		}
	}
	//*reply = "La stringa inserita è stata memorizzata con successo. La sua chiave è: " + strconv.Itoa(key)
	//fmt.Println(curr_node.Resources)
	return nil
}

func (t *RegisterNode) UpdateSuccessorAfterRemove(node Node, reply *bool) error {

	//fmt.Println("curr_nodo:", curr_node.ID)

	//aggiorno il successore del predeccesore del nodo che vado a eliminare
	curr_node.Successor = node.Successor

	//fmt.Println("id new_successor_curr_nodo:", curr_node.Successor)

	return nil
}

func (t *RegisterNode) UpdateSuccessorResources(node Node, reply *bool) error {

	// Copia le risorse dal nodo corrente al suo successore
	for id, resource := range node.Resources {
		curr_node.Resources[id] = resource
	}
	//fmt.Println("curr_nodo:", curr_node.ID)

	//aggiorno il predecessore del successore del nodo che vado a eliminare
	curr_node.Predecessor = node.Predecessor
	//fmt.Println("id new_predecessor_curr_nodo:", curr_node.Predecessor)
	//fmt.Println("risorse:", curr_node.Resources)

	return nil
}

func (t *RegisterNode) InitiateLeave(curr_nodeIP string, reply *bool) error {
	// Implementa qui la logica per il leave volontario del nodo
	// Assicurati di trasferire i dati, aggiornare le tabelle di routing, ecc.

	if curr_node.Ip == curr_nodeIP {
		fmt.Println("Processo di leave avviato.")
		fmt.Println("Il mio successore è:", curr_node.Successor)

		client, err := rpc.DialHTTP("tcp", curr_node.Successor)
		if err != nil {
			log.Fatal("Client connection error: ", err)
		}
		err = client.Call("RegisterNode.UpdateSuccessorResources", curr_node, &reply)
		if err != nil {
			log.Fatal("Errore durante l'update delle risorse del successore:", err)
		}

		// Rimuovi le risorse copiate dal nodo
		for id := range curr_node.Resources {
			fmt.Println("Trasferisco risorsa con id:", id)
			delete(curr_node.Resources, id)
		}

		client, err = rpc.DialHTTP("tcp", curr_node.Predecessor)
		if err != nil {
			log.Fatal("Client connection error: ", err)
		}
		err = client.Call("RegisterNode.UpdateSuccessorAfterRemove", curr_node, &reply)
		if err != nil {
			log.Fatal("Errore durante l'update delle risorse del successore:", err)
		}
		fmt.Println("Bye!Bye!")
	}

	*reply = true

	close(stopChan) //chiudi connessione

	return nil
}

func startFingerTableUpdater(interval time.Duration, node *Node, num_bit int, stopChan <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			getSuccNode(node)
			getPrecNode(node)
			InitializeFingerTable(node, num_bit)
		case <-stopChan:
			fmt.Printf("Connessione interrotta correttamente.\n")
			os.Exit(0)
		}
	}
}

func getSuccNode(node *Node) error {

	var reply *string
	client, err := rpc.DialHTTP("tcp", serverAddr)
	if err != nil {
		log.Fatal("Client connection error: ", err)
	}
	err = client.Call("Manager.GetSuccessorNode", node.ID, &reply)
	if err != nil {
		log.Fatal("Errore durante l'update delle risorse del successore:", err)
	}

	node.Successor = *reply

	client.Close()

	return nil
}
func getPrecNode(node *Node) error {

	var reply *string
	client, err := rpc.DialHTTP("tcp", serverAddr)
	if err != nil {
		log.Fatal("Client connection error: ", err)
	}
	err = client.Call("Manager.GetPredecessorNode", node.ID, &reply)
	if err != nil {
		log.Fatal("Errore durante l'update delle risorse del successore:", err)
	}

	node.Predecessor = *reply

	client.Close()

	return nil
}

func main() {

	/*args := os.Args
	if len(args) < 2 {
		fmt.Println("Error: Not enough arguments. Enter the node's IP address:port number.")
		os.Exit(1)
	}*/

	// Ottengo il nome host del container
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Errore nell'ottenere il nome host:", err)
		return
	}
	// Ottiengo la porta e il numero di bit dalle variabili d'ambiente
	port := os.Getenv("NODE_PORT")
	num_bit_string := os.Getenv("BIT")
	num_b, err := strconv.Atoi(num_bit_string)
	num_bit := int(math.Pow(2, float64(num_b)))
	if err != nil {
		log.Fatal("Errore nella conversione per il num_bit:", err)
	}

	addr, err := net.LookupHost(hostname)
	if err != nil {
		log.Fatal("Errore nel ottenere l'indirizzo ip dell'host:", err)
	}
	cleanedIPAddress := strings.Trim(addr[0], "[]")
	// Unisco lindirizzo ip e la porta in una sola stringa
	address := fmt.Sprintf("%s:%s", cleanedIPAddress, port)
	//num_bit := 32
	//chord_node := NewNode(args[1], num_bit)

	//creo il nodo
	chord_node := NewNode(address, num_bit)

	//vado a comunicare al mio successore che deve aggiornare il suo predecessore
	if chord_node.Successor != chord_node.Ip {
		client, err := rpc.DialHTTP("tcp", chord_node.Successor)
		if err != nil {
			log.Fatal("dialing:", err)
		}

		err = client.Call("RegisterNode.UpdatePred", chord_node, &(chord_node.Resources))
		if err != nil {
			log.Fatal("Client invocation error: ", err)
		}
	}

	//vado a comunicare al mio predecessore che deve aggiornare il suo successore e la sua finger table
	if chord_node.Predecessor != chord_node.Ip {
		client, err := rpc.DialHTTP("tcp", chord_node.Predecessor)
		if err != nil {
			log.Fatal("dialing:", err)
		}

		err = client.Call("RegisterNode.UpdateSucc", chord_node, &(chord_node.Resources))
		if err != nil {
			log.Fatal("Client invocation error: ", err)
		}
	}

	n := int(math.Log2(float64(num_bit)))
	InitializeFingerTable(&chord_node, n)

	registeredNode := new(RegisterNode)
	rpc.Register(registeredNode)
	rpc.HandleHTTP()

	//se il nodo esiste allora la eseguo altrimenti
	go startFingerTableUpdater(20*time.Second, &chord_node, num_b, stopChan)

	listener, err := net.Listen("tcp", chord_node.Ip)
	if err != nil {
		log.Fatal("Listener error: ", err)
	}
	http.Serve(listener, nil)
	if err != nil {
		log.Fatal("Errore nel server HTTP: ", err)
	}

}
