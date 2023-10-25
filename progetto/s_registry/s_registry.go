package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type Node struct {
	ID            int
	Predecessor   string
	Successor     string
	Resources     map[int](string)
	Ip            string
	fingerTable   []int
	fingerTableIP map[int]string // Inizializzazione della mappa per gli IP
}

var nodeMap = make(map[int](string))

type Manager string

var node Node

var nodeList = make([]int, 0)

// registro il nodo ritornandoli il successore e il predecessore
func (m *Manager) RegisterNode(current_node Node, reply *Node) error {

	var nodeListA = make([]int, 0) //lista temp utilizzata per aggiornare la nodeList

	//verifico se l'id del current_node esiste già nella mappa
	_, exists := nodeMap[current_node.ID]
	if !exists {
		nodeMap[current_node.ID] = current_node.Ip
	} else {
		err := errors.New("Nodo già esistente")
		fmt.Println(err)
		return err
	}

	var position int
	if len(nodeList) == 0 {
		nodeList = append(nodeList, current_node.ID)

		position = 0
		reply.Successor = nodeMap[nodeList[(position)%len(nodeList)]]
		reply.Predecessor = nodeMap[nodeList[(position)%len(nodeList)]]

	} else if len(nodeList) == 1 {
		if current_node.ID > nodeList[0] {
			nodeList = append(nodeList, current_node.ID)
			position = 1
		} else {
			nodeListA = append(nodeListA, current_node.ID)
			nodeList = append(nodeListA, nodeList[0])
			position = 0
		}
		reply.Successor = nodeMap[nodeList[(position+1)%len(nodeList)]]
		reply.Predecessor = nodeMap[nodeList[(position+1)%len(nodeList)]]

	} else {

		//controllo se non esiste già un nodo che si cerca di aggiungere. Se esiste mi restituisce la posizione
		position := -1
		for i, v := range nodeList {
			if v == current_node.ID {
				position = i
				break
			}
		}
		if position != -1 {
			fmt.Printf("Nodo già presente.")
		} else {
			for i := 1; i < len(nodeList); i++ {
				if current_node.ID > nodeList[len(nodeList)-1] {
					position = len(nodeList)
					break
				} else if current_node.ID < nodeList[i] && current_node.ID > nodeList[i-1] {
					position = i
					break
				} else {
					position = len(nodeList) - 1 - i
				}
			}
			nodeListA = append(nodeListA, nodeList[:position]...)
			nodeListA = append(nodeListA, current_node.ID)
			nodeList = append(nodeListA, nodeList[position:]...)

		}
		reply.Successor = nodeMap[nodeList[(position+1)%len(nodeList)]]
		reply.Predecessor = nodeMap[nodeList[((position-1)+len(nodeList))%len(nodeList)]]

	}

	for key, value := range nodeMap {
		fmt.Printf("Nodo: %d, Ip address: %s\n", key, value)
	}
	fmt.Println()

	return nil

}

var currentRoundRobinIndex int = 0 // Variabile globale per tenere traccia dello stato del round-robin

// politica di round-robin per la selezione dei nodi in modo sequenziale
func (m *Manager) ContactNode(arg string, reply *string) error {

	if len(nodeList) == 0 {
		err := errors.New("Non sono presenti nodi nel sistema.")
		fmt.Println(err)
		reply = nil
	} else {
		// Calcola l'indice del nodo da selezionare in base al round-robin
		indexToSelect := currentRoundRobinIndex % len(nodeList)
		*reply = nodeMap[nodeList[indexToSelect]]

		//fmt.Println("Nodo contattato per inserimento: ", nodeMap[nodeList[indexToSelect]])
		// Incrementa l'indice per la prossima chiamata, in modo che il successivo sia selezionato
		currentRoundRobinIndex++
	}
	return nil
}

func (m *Manager) GetSuccessorNode(nodeId int, reply *string) error {
	for i := 0; i < len(nodeList); i++ {
		if nodeList[i] == nodeId {
			// Trovato il nodo con l'ID dato
			if i+1 < len(nodeList) {
				// Se non siamo all'ultimo elemento, il successore è l'elemento successivo
				*reply = nodeMap[nodeList[i+1]]
			} else {
				// Se siamo all'ultimo elemento, il successore è il primo elemento (lista circolare)
				*reply = nodeMap[nodeList[0]]
			}
			//return nil
		}
	}

	return nil
}

func (m *Manager) GetPredecessorNode(nodeId int, reply *string) error {
	for i := 0; i < len(nodeList); i++ {
		if nodeList[i] == nodeId {
			// Trovato il nodo con l'ID dato
			if i > 0 {
				// Se non siamo al primo elemento, il predecessore è l'elemento precedente
				*reply = nodeMap[nodeList[i-1]]
			} else {
				// Se siamo al primo elemento, il predecessore è l'ultimo elemento (lista circolare)
				*reply = nodeMap[nodeList[len(nodeList)-1]]
			}
			return nil
		}
	}

	return nil
}

// RequestLeave gestisce la richiesta di leave inviata dal client
func (m *Manager) RequestLeave(nodeID int, reply *bool) error {

	var node_ip string

	if _, exists := nodeMap[nodeID]; exists {
		// L'ID esiste nella mappa nodeMap
		// Esegui le azioni necessarie quando l'ID esiste
		node_ip = nodeMap[nodeID]
	} else {
		// L'ID non esiste nella mappa nodeMap
		//fmt.Println("Nodo non presente nella rete.")
		return nil
	}

	client, err := rpc.DialHTTP("tcp", node_ip)
	if err != nil {
		log.Fatal("Client connection error: ", err)
	}
	err = client.Call("RegisterNode.InitiateLeave", node_ip, &reply)
	if err != nil {
		fmt.Println("Errore durante l'iniziazione del leave:", err)
		*reply = false
		return err
	}

	// Il leave è stato avviato con successo
	*reply = true

	//rimuovo l'elemento dalla nodeList e dalla nodeMap
	for i := 0; i < len(nodeList); i++ {
		if nodeList[i] == nodeID {
			// Trovato l'elemento da rimuovere
			// Rimuovo l'elemento utilizzando append
			nodeList = append(nodeList[:i], nodeList[i+1:]...)
			break // Esci dal ciclo una volta rimosso l'elemento
		}
	}
	delete(nodeMap, nodeID)

	fmt.Println()
	for key, value := range nodeMap {
		fmt.Printf("Nodo: %d, Ip address: %s\n", key, value)
	}

	return nil
}

func main() {

	manager := new(Manager)
	rpc.Register(manager)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer listener.Close()
	http.Serve(listener, nil)

}
