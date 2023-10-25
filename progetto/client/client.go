package main

import (
	"bufio"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
)

var serverAddr = "0.0.0.0:8000"

func main() {

	var selectedOption int
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Scegli una fra le seguenti opzioni: ")
	fmt.Println("1-Inserire una stringa")
	fmt.Println("2-Recuperare una stringa")
	fmt.Println("3-Avviare il processo di uscita per un nodo")
	fmt.Println()

	for {
		var reply string
		_, err := fmt.Scanln(&selectedOption)
		if err != nil {
			fmt.Println("Opzione non valida. Inserisci 1, 2 o 3.")
			scanner.Scan()
			continue
		}

		switch selectedOption {
		case 1:
			var inputString string
			fmt.Println("Hai scelto l'opzione 1. Qual è la stringa che vuoi inserire?")
			scanner.Scan()
			inputString = scanner.Text()

			//contatto il registry
			client, err := rpc.DialHTTP("tcp", serverAddr)
			if err != nil {
				log.Fatal("dialing:", err)
			}

			err = client.Call("Manager.ContactNode", inputString, &reply)
			if err != nil {
				log.Fatal("Client invocation error: ", err)
			}

			//contatto il nodo ritornato dal registry
			client, err = rpc.DialHTTP("tcp", reply)
			if err != nil {
				log.Fatal("Client connection error: ", err)
			}

			err = client.Call("RegisterNode.AddData", inputString, &reply)
			if err != nil {
				log.Fatal("Client invocation error: ", err)
			}

			fmt.Println(reply)

		case 2:
			var key int
			fmt.Println("Hai scelto l'opzione 2. Qual è la chiave della risorsa ricercata?")
			scanner.Scan()
			key, err := strconv.Atoi(scanner.Text())
			if err != nil {
				fmt.Println("Inserire un numero valido.")
				return

			} else {

				//contatto il registry
				client, err := rpc.DialHTTP("tcp", serverAddr)
				if err != nil {
					log.Fatal("Dialing:", err)
				}

				err = client.Call("Manager.ContactNode", strconv.Itoa(key), &reply)
				if err != nil {
					log.Fatal("Client invocation error: ", err)
				}

				//contatto il nodo
				client, err = rpc.DialHTTP("tcp", reply)
				if err != nil {
					log.Fatal("Dialing: ", err)
				}

				err = client.Call("RegisterNode.GetData", &key, &reply)
				if err != nil {
					log.Fatal("Client invocation error: ", err)
				}

				if reply == "" {
					fmt.Printf("Nessuna stringa corrispondente alla chiave ricercata")
				} else {
					fmt.Printf("La stringa memorizzata con la chiave %d è: %s", key, reply)
				}
			}

		case 3:

			var nodoID int
			fmt.Println("Hai scelto l'opzione 3. Qual è l'ID del nodo che deve avviare la leave?")
			scanner.Scan()
			nodoID, err := strconv.Atoi(scanner.Text())
			if err != nil {
				fmt.Println("Errore di conversione. Devi inserire un numero valido.")
				break

			}

			//contatto il registry
			client, err := rpc.DialHTTP("tcp", serverAddr)
			if err != nil {
				log.Fatal("Dialing:", err)
			}

			// Chiamata RPC per richiedere il leave di un nodo
			var reply bool
			err = client.Call("Manager.RequestLeave", nodoID, &reply)
			if err != nil {
				fmt.Println("Client invocation error: ", err)
				return
			}

			if reply {
				fmt.Println("Richiesta di leave inviata con successo al server.")
			} else {
				fmt.Println("La richiesta di leave non è riuscita. Nodo non presente.")
			}

		default:
			fmt.Println("Opzione non presente. Non è stato inserito un numero corretto.")
		}

		fmt.Println()
		fmt.Println("Scegli una nuova opzione")
	}
}
