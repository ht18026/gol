package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

func buildWorld(world [][]byte, index, subHeight, imageWidth int,numOfWorker int) [][]byte {

	//make a new world with height of workers' world plus 2
	workPlace := make([][]byte,subHeight + 2)
	for i := range workPlace{
		workPlace[i] = make([]byte, imageWidth)
	}

	//create a switch sentence to deal with the cases that height out of boundary
	switch index{
	//height of -1
	case 0:
		workPlace[0] = world[len(world) - 1]
		for y := 1; y < subHeight + 2; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * subHeight + y - 1][x]
			}
		}

	//height of imageHeight + 1
	case numOfWorker - 1:
		workPlace[subHeight + 1] = world[0]
		for y := 0; y < subHeight + 1; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * subHeight + y - 1][x]
			}
		}

	//height in the boundary
	default:
		for y := 0; y < subHeight + 2; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * subHeight + y - 1][x]

			}
		}
	}

	return workPlace                                           //return the cut world to be sent to workers
}

var m sync.Mutex                                               //define a mutex lock out of the func body
//a function that deal with every single world parts and return the next turn of this part
func worker(subHeight, imageWidth int, workPlace [][]byte, initialWorld [][]byte,numOfWorker int,done func()) {        //using the waitgroup to let main wait all workers
	defer done()
	newWorldPart := make([][]byte, subHeight+2)
	for i := range newWorldPart{
		newWorldPart[i] = make([]byte, imageWidth)
	}

	newWorldPart = schrodinger(workPlace)                 //do the judgement
	//return newWorldPart
    m.Lock()                                              //using mutex lock to protect the critical section
	for y := 1; y < subHeight+1; y++{
			initialWorld[subHeight*numOfWorker+y-1] = newWorldPart[y] //access to memory
	}
	m.Unlock()                                             // unlock, leave the critical section





}

//the logic of the Game Of Live
//i call it schrodinger because if you dont observe the cat(which is the world containing cells),
// u will never know whether it is alive
func schrodinger(cat [][]byte)[][]byte{
	imageHeight := len(cat)
	imageWidth := len(cat[0])
	nextWorld := make([][]byte, imageHeight)
	for i := range nextWorld {
		nextWorld[i] = make([]byte, imageWidth)
	}
	//create a for loop go through all cells in the world
	for y := 1; y < imageHeight - 1; y++ {
		for x := 0; x < imageWidth; x++ {
			//create a int value that counts how many alive neighbours does a cell have
			aliveNeighbours := 0
			//extract the 3x3 matrix which centred at the cell itself
			//go through every neighbour and count the aliveNeighbours
			for i := -1; i < 2; i++{
				for j := -1; j < 2; j++{
					if i == 0 && j == 0{continue}                                              //I don't care if the cell itself is alive or dead at this stage
					if cat[y + i][(x + j + imageWidth) % imageWidth] == 255{                  //if there is an alive neighbour, the count of alive neighbours increase by 1
						aliveNeighbours += 1
					}
				}
			}
			if cat[y][x] == 255{
				if aliveNeighbours < 2 || aliveNeighbours > 3{                  //if the cell itself is alive, check the neighbours:
					nextWorld[y][x] = 0                                         //if it has <2 or>3 alive neighbours, it will die in nextWorld :(
				} else{nextWorld[y][x] = 255}                                   //if it has =2 or =3 alive neighbours, it will survive in nextWorld :)
			}
			if cat[y][x] == 0{
				if aliveNeighbours == 3{                                        //if the cell itself is dead, check the neighbours:
					nextWorld[y][x] = 255                                       //if it has =3 neighbours, it will become alive in nextWorld ;)
				}else{nextWorld[y][x] = 0}
			}
		}
	}

	return nextWorld
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}


	// Calculate the new state of Game of Life after the given number of turns.
	//calculate the height of the world every worker get

	subHeight := p.imageHeight/p.threads
	for turns := 0; turns < p.turns; turns++ {


		//copy the world and distribute them to workers
        newWorld := make([][]byte,p.imageHeight)                              //using equal will fail because slice pass address
        for i := range newWorld{
        	newWorld[i] = make([]byte,p.imageWidth)
		}
        copy(newWorld,world)

        var wg sync.WaitGroup
        wg.Add(p.threads)                                                    //using counter count the num of worker
		for i := 0; i < p.threads; i++{


			  go   worker(subHeight, p.imageWidth,buildWorld(world,i,subHeight,p.imageWidth,p.threads),newWorld,i,wg.Done)   //in workers they give the new world


		}
		wg.Wait()                                                       //wait for workers
		//receive world parts and re-gather them
		world = newWorld
	}


	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.outputWorld <- world
	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}