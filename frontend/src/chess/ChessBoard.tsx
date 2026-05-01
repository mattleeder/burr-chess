import React, { useContext, useEffect, useRef, useState } from "react";
import { PieceColour, PieceVariant } from "./ChessLogic";
import { GameContext, gameContext } from "./GameContext";
import { API } from "../api";
import "./ChessBoard.css";

const colourToString = new Map<PieceColour, string>()

colourToString.set(PieceColour.White, 'white')
colourToString.set(PieceColour.Black, 'black')

export const variantToString = new Map<PieceVariant, string>()

variantToString.set(PieceVariant.Pawn, 'pawn')
variantToString.set(PieceVariant.Knight, 'knight')
variantToString.set(PieceVariant.Bishop, 'bishop')
variantToString.set(PieceVariant.Rook, 'rook')
variantToString.set(PieceVariant.Queen, 'queen')
variantToString.set(PieceVariant.King, 'king')

enum ClickAction {
  clear,
  showMoves,
  makeMove,
  choosePromotion,
}

interface Rect {
  top: number,
  left: number,
  width: number,
  height: number,
}

function getSquareIdxFromClick(x: number, y: number, rect?: React.RefObject<Rect | null>) {
  if (rect === undefined || rect === null || rect.current === null) {
    throw new Error("Bounding rect for board is not defined")
  }

  const squareWidth = rect.current.width / 8

  const boardXPosition = Math.floor((x - rect.current.left) / squareWidth)
  const boardYPosition = Math.floor((y - rect.current.top) / squareWidth)
  const position =  boardYPosition * 8 + boardXPosition
  return position
}

async function clickHandler(
  position: number,
  game: gameContext,
  promotionActive: boolean,
  promotionSquare: number,
  moves: number[],
  setMoves: React.Dispatch<React.SetStateAction<number[]>>,
  captures: number[],
  setCaptures: React.Dispatch<React.SetStateAction<number[]>>,
  selectedPiece: number | null,
  setSelectedPiece: React.Dispatch<React.SetStateAction<number | null>>,
  promotionNextMove: boolean,
  setPromotionNextMove: React.Dispatch<React.SetStateAction<boolean>>,
  setPromotionActive: React.Dispatch<React.SetStateAction<boolean>>,
  setPromotionSquare: React.Dispatch<React.SetStateAction<number>>,
  waiting: boolean,
  setWaiting: React.Dispatch<React.SetStateAction<boolean>>,
) {
  if (game.flip) {
    position = 63 - position
  }

  // If waiting do nothing
  if (waiting) {
    return
  }

  // Set Waiting
  setWaiting(true)

  // Get clickAction state
  let clickAction = ClickAction.clear
  if (game?.matchData.activeMove != (game as gameContext)?.matchData.stateHistory.length - 1) {
    setWaiting(false)
    return
  } else if (game.matchData.gameOverStatus != 0) {
    setWaiting(false)
    return
  } else if (promotionActive && [0, 8, 16, 24].includes(Math.abs(position - promotionSquare))) {
    clickAction = ClickAction.choosePromotion
  } else if ([...moves, ...captures].includes(position)) {
    clickAction = ClickAction.makeMove
    // @TODO
    // null == null 
  } else if (game?.matchData.activeState.board[position][0] == game?.playerColour && position != selectedPiece) {
    clickAction = ClickAction.showMoves
  }

  // If showing promotion, check promotion selection, if promoting submit move and reset to bare

  // If showing moves, check for move click, if move clicked, submit and reset to bare

  // If clicked on piece, select piece

  // If clicked on board, reset to bare

  switch (clickAction) {
  case ClickAction.clear:
    setToBare(
      setSelectedPiece,
      setMoves,
      setCaptures,
      setPromotionNextMove,
      setPromotionActive,
      setPromotionSquare
    )
    break;

  case ClickAction.makeMove:
  case ClickAction.choosePromotion:
  {
    if (selectedPiece === null) {
      throw new Error("Posting move with no piece")
    }

    // Render promotion component
    if (promotionNextMove) {
      setPromotionActive(true)
      setPromotionSquare(position)
      setPromotionNextMove(false)
      break;
    }

    let promotionString = ""
    if (clickAction == ClickAction.choosePromotion) {
      const promotionIndex = [0, 8, 16, 24].indexOf(Math.abs(position - promotionSquare))
      promotionString = "qnrb"[promotionIndex]
      position = promotionSquare
    }
    // Send current FEN, piece, move, new FEN
    wsPostMove(position, selectedPiece, promotionString, game)

    // Clear cache, clear moves
    setToBare(
      setSelectedPiece,
      setMoves,
      setCaptures,
      setPromotionNextMove,
      setPromotionActive,
      setPromotionSquare
    )
    break;
  }

  case ClickAction.showMoves:
  // Fetch moves
  {
    const data = await fetchPossibleMoves(position, game)

    // Set Moves
    setMoves(data["moves"] || [])
    setCaptures(data["captures"] || [])
    setPromotionNextMove(data["triggerPromotion"])

    // Show Moves
    setSelectedPiece(position)
    break;
  }
  }

  setWaiting(false)
}

function setToBare(
  setSelectedPiece: React.Dispatch<React.SetStateAction<number | null>>,
  setMoves: React.Dispatch<React.SetStateAction<number[]>>,
  setCaptures: React.Dispatch<React.SetStateAction<number[]>>,
  setPromotionNextMove: React.Dispatch<React.SetStateAction<boolean>>,
  setPromotionActive: React.Dispatch<React.SetStateAction<boolean>>,
  setPromotionSquare: React.Dispatch<React.SetStateAction<number>>,
) {
  setSelectedPiece(null)
  setMoves([])
  setCaptures([])
  setPromotionNextMove(false)
  setPromotionActive(false)
  setPromotionSquare(0)
}

async function fetchPossibleMoves(position: number, game: gameContext) {
  try {
    const mostRecentMove = game.matchData.stateHistory.at(-1)
    if (!mostRecentMove) {
      return
    }
    const response = await fetch(API.fetchMoves, {
      "method": "POST",
      "body": JSON.stringify({
        "fen": mostRecentMove["FEN"],
        "piece": position,
      })
    })

    if (!response.ok) {
      throw new Error(`Response status: ${response.status}`)
    }

    const data = await response.json()
    return data
  }

  catch (error: unknown) {
    if (error instanceof Error) {
      console.error(error.message)
    } else {
      console.error(error)
    }
  }

  return {}

}

function wsPostMove(position: number, piece: number, promotion: string, game: gameContext) {
  game?.webSocket.current?.send(JSON.stringify({
    "messageType": "postMove",
    "body": {
      "piece": piece,
      "move": position,
      "promotionString": promotion,
    }
  }))
}

function getRowAndColFromBoardIndex(idx: number, flip: boolean): [number, number] {
  const row = Math.floor(idx / 8)
  const col = idx % 8

  // Checks if the board needs to be flipped and transforms the row and col
  if (flip) {
    return [Math.abs(7 - row), Math.abs(7 - col)]
  }
  return [row, col]
}

// function PiecesComponent({ flip, squareWidth, onDragEnd, rect, board }: { flip: boolean, squareWidth: number, onDragEnd: (startIdx: number, endIdx: number) => void, rect?: React.RefObject<Rect | null>, board: [PieceColour | null, PieceVariant | null][] }) {

//   return (
//     board.map((square, idx) => {
//       const [colour, variant] = square
//       if (colour === null || variant === null) {
//         return <React.Fragment key={idx} />
//       }

//       // TODO: dragging
//       // On mousedown record position
//       // On mouseup record position
//       // If squares are different, try to post that move
      
//       const [row, col] = getRowAndColFromBoardIndex(idx, flip)
//       return (
//         <div 
//           key={idx}
//           draggable={true}
//           onDragEnd={(event) => {
//             console.log(`onDragEnd ${event.clientX}, ${event.clientY}`)
//             const endPosition = getSquareIdxFromClick(event.clientX, event.clientY, rect)
//             onDragEnd(idx, endPosition)
//           }}
//           className={`${colourToString.get(colour)}-${variantToString.get(variant)} pieceTransition`} 
//           style={{ 
//             transform: `translate(${col * squareWidth}px, ${row * squareWidth}px)`,
//             width: `${squareWidth}px`,
//             height: `${squareWidth}px`,
//             backgroundSize: `${squareWidth}px`,
//           }} 
//         />
//       )
//     })
//   )
// }

function PiecesComponent({ flip, squareWidth, onDragEndCallback, rect, colour, variant, index }: { flip: boolean, squareWidth: number, onDragEndCallback: (startIdx: number, endIdx: number) => void, rect?: React.RefObject<Rect | null>, colour: PieceColour | null, variant: PieceVariant | null, index: number }) {
  
  if (colour === null || variant === null) {
    return <React.Fragment key={index} />
  }
  const [row, col] = getRowAndColFromBoardIndex(index, flip)

  return (
    <div
      key={`${colourToString.get(colour)}-${variantToString.get(variant)}`}
      draggable={true}
      onDragEnd={(event) => {
        const endPosition = getSquareIdxFromClick(event.clientX, event.clientY, rect)
        onDragEndCallback(index, endPosition)
      }}
      className={`${colourToString.get(colour)}-${variantToString.get(variant)} pieceTransition`} 
      style={{
        position: "absolute",
        transform: `translate(${col * squareWidth}px, ${row * squareWidth}px)`,
        width: `${squareWidth}px`,
        height: `${squareWidth}px`,
        backgroundSize: `${squareWidth}px`,
        transition: "transform 1s",
      }} 
    />
  )
}

function MovesComponent({ moves, flip, squareWidth }: { moves: number[], flip: boolean, squareWidth: number }) {
  return (
    (moves).map((move, idx) => {
      const [row, col] = getRowAndColFromBoardIndex(move, flip)
      return (
        <div 
          key={idx} 
          className='potential-move' 
          style={{ 
            transform: `translate(${col * squareWidth}px, ${row * squareWidth}px)`,
            width: `${squareWidth}px`,
            height: `${squareWidth}px`,
            backgroundSize: `${squareWidth}px`,
          }} 
        />
      )
    }))
}

function CapturesComponent({ captures, flip, squareWidth }: { captures: number[], flip: boolean, squareWidth: number }) {
  return (
    captures.map((move, idx) => {
      const [row, col] = getRowAndColFromBoardIndex(move, flip)
      return (
        <div 
          key={idx} 
          className='potential-capture' 
          style={{ 
            transform: `translate(${col * squareWidth}px, ${row * squareWidth}px)`,
            width: `${squareWidth}px`,
            height: `${squareWidth}px`,
            backgroundSize: `${squareWidth}px`,
          }} 
        />
      )
    }))
}

function LastMoveComponent({ flip, squareWidth, lastMove, showLastMove }: { flip: boolean, squareWidth: number, lastMove: [number, number], showLastMove: boolean }) {
  return (
    lastMove.map((move, idx) => {
      if (!showLastMove) {
        return <React.Fragment key={idx} />
      }
      const [row, col] = getRowAndColFromBoardIndex(move, flip)

      // @TODO: sort out lastMove Border radius with flips
      return (
        <div 
          key={idx} 
          className='last-move' 
          style={{ 
            transform: `translate(${col * squareWidth}px, ${row * squareWidth}px)`,
            width: `${squareWidth}px`,
            height: `${squareWidth}px`,
            backgroundSize: `${squareWidth}px`,
            borderTopLeftRadius: `${idx == 0 ? 4 : 0}px`,
            borderTopRightRadius: `${idx == 7 ? 4 : 0}px`,
            borderBottomLeftRadius: `${idx == 56 ? 4 : 0}px`,
            borderBottomRightRadius: `${idx == 63 ? 4 : 0}px`,
          }} 
        />
      )
    }))
}

function PromotionComponent({ promotionSquare, promotionActive, flip, squareWidth }: { promotionSquare: number, promotionActive: boolean, flip: boolean, squareWidth: number }) {
  if (!promotionActive) {
    return <></>
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_row, col] = getRowAndColFromBoardIndex(promotionSquare, flip)

  let promotionColour = colourToString.get(PieceColour.Black)

  if (promotionSquare <= 7) {
    promotionColour = colourToString.get(PieceColour.White)
  }

  return (
    <>
      <div 
        className={`${promotionColour}-queen promotion`} 
        style={{ 
          transform: `translate(${col * squareWidth}px, ${0 * squareWidth}px)`,
          width: `${squareWidth}px`,
          height: `${squareWidth}px`,
          backgroundSize: `${squareWidth}px`,
        }} 
      />

      <div 
        className={`${promotionColour}-knight promotion`} 
        style={{ 
          transform: `translate(${col * squareWidth}px, ${1 * squareWidth}px)`,
          width: `${squareWidth}px`,
          height: `${squareWidth}px`,
          backgroundSize: `${squareWidth}px`,
        }} 
      />

      <div 
        className={`${promotionColour}-rook promotion`} 
        style={{ 
          transform: `translate(${col * squareWidth}px, ${2 * squareWidth}px)`,
          width: `${squareWidth}px`,
          height: `${squareWidth}px`,
          backgroundSize: `${squareWidth}px`,
        }} 
      />

      <div 
        className={`${promotionColour}-bishop promotion`} 
        style={{ 
          transform: `translate(${col * squareWidth}px, ${3 * squareWidth}px)`,
          width: `${squareWidth}px`,
          height: `${squareWidth}px`,
          backgroundSize: `${squareWidth}px`,
        }} 
      />
    </>
  )
}

function GameOverComponent({ squareWidth }: { squareWidth: number }) {
  const game = useContext(GameContext)
  if (!game) {
    throw new Error("LastMoveComponent must be called from within a GameContext")
  }

  if (game.matchData.gameOverStatus == 0) {
    return <></>
  }

  const gameOverStatusCodes = ["Ongoing", "Stalemate", "Checkmate", "Threefold Repetition", "Insufficient Material", "White Flagged", "Black Flagged", "Draw", "White Resigned", "Black Resigned", "Game Aborted", "White Disconnected", "Black Disconnected"]
  const gameOverText = gameOverStatusCodes[game?.matchData.gameOverStatus || 0]

  return <div style={{ transform: `translate(${0}px, ${squareWidth * 4}px)`, color: "black" }}>{gameOverText}</div>
}

function getRect(top: number, left: number, width: number, height: number): Rect {
  return {
    top: top,
    left: left,
    width: width,
    height: height,
  }
}

export function ChessBoard({ resizeable, defaultWidth, chessboardContainerStyles, enableClicking }: { resizeable: boolean, defaultWidth: number, chessboardContainerStyles?: React.CSSProperties, enableClicking: boolean }) {
  const boardRef = useRef<HTMLDivElement | null>(null)
  const rect = useRef<Rect | null>(null)
  
  const [waiting, setWaiting] = useState(false)
  
  const [selectedPiece, setSelectedPiece] = useState<number | null>(null)
  const [moves, setMoves] = useState<number[]>([])
  const [captures, setCaptures] = useState<number[]>([])
  
  const [promotionNextMove, setPromotionNextMove] = useState(false)
  const [promotionActive, setPromotionActive] = useState(false)
  const [promotionSquare, setPromotionSquare] = useState(0)
  const [boardWidth, setBoardWidth] = useState(defaultWidth)

  const game = useContext(GameContext)
  if (!game) {
    throw new Error('ChessBoard must be used within a GameContext Provider');
  }


  useEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const boundingRect = entry.target.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.width)
      }
    })
    if (boardRef.current) {
      resizeObserver.observe(boardRef.current)
    }

    window.addEventListener('scroll', () => {
      if (boardRef.current) {
        const boundingRect = boardRef.current.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.width)
      }
    })

    window.addEventListener('resize', () => {
      if (boardRef.current) {
        const boundingRect = boardRef.current.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.width)
      }
    })

    return () => {
      resizeObserver.disconnect()

      window.removeEventListener('scroll', () => {
        if (boardRef.current) {
          const boundingRect = boardRef.current.getBoundingClientRect()
          rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
          setBoardWidth(boundingRect.width)
        }
      })

      window.removeEventListener('resize', () => {
        if (boardRef.current) {
          const boundingRect = boardRef.current.getBoundingClientRect()
          rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
          setBoardWidth(boundingRect.width)
        }
      })

    }
  }, [boardRef])
  
  // Clear board when changing active move
  useEffect(() => {
    setToBare(
      setSelectedPiece,
      setMoves,
      setCaptures,
      setPromotionNextMove,
      setPromotionActive,
      setPromotionSquare
    )
  }, [game.matchData.activeMove])

  const squareWidth = boardWidth / 8

  const chessboardContainerStyle = {...chessboardContainerStyles}
  chessboardContainerStyle["width"] = `${boardWidth}px`
  chessboardContainerStyle["height"] = `${boardWidth}px`
  if (resizeable) {
    chessboardContainerStyle["resize"] = "both"
    chessboardContainerStyle["overflow"] = "auto"
  }
  
  return (
    <div className="chessboard-container" style={chessboardContainerStyle} ref={boardRef}>
      <div 
        className='chessboard' 
        style={{
          width: `${boardWidth}px`,
          height: `${boardWidth}px`,
          backgroundSize: `${boardWidth}px`,
        }} 
        onMouseDown={(event) => {
          const position = getSquareIdxFromClick(event.clientX, event.clientY, rect)
          if (enableClicking) {
            clickHandler(
              position,
              game,
              promotionActive,
              promotionSquare,
              moves,
              setMoves,
              captures,
              setCaptures,
              selectedPiece,
              setSelectedPiece,
              promotionNextMove,
              setPromotionNextMove,
              setPromotionActive,
              setPromotionSquare,
              waiting,
              setWaiting
            )}}}
      >
        <LastMoveComponent flip={game.flip} squareWidth={squareWidth} lastMove={game.matchData.activeState.lastMove} showLastMove={game.matchData.activeMove != 0}/>
        {game.matchData.activeState.board.map((square, idx) => {
          const [colour, variant] = square
          return (
            <PiecesComponent flip={game.flip} squareWidth={squareWidth} rect={rect} colour={colour} variant={variant} index={idx} onDragEndCallback={
              (startIdx: number, endIdx: number) => {
                if (startIdx != endIdx) {
                  clickHandler(
                    endIdx,
                    game,
                    promotionActive,
                    promotionSquare,
                    moves,
                    setMoves,
                    captures,
                    setCaptures,
                    selectedPiece,
                    setSelectedPiece,
                    promotionNextMove,
                    setPromotionNextMove,
                    setPromotionActive,
                    setPromotionSquare,
                    waiting,
                    setWaiting
                  )
                }
              }
            }/>
          )
        })}
        <MovesComponent moves={moves} flip={game.flip} squareWidth={squareWidth}/>
        <CapturesComponent captures={captures} flip={game.flip} squareWidth={squareWidth}/>
        <PromotionComponent promotionSquare={promotionSquare} promotionActive={promotionActive} flip={game.flip} squareWidth={squareWidth}/>
        <GameOverComponent squareWidth={squareWidth}/>
      </div>
    </div>
  )
}

export function FrozenChessBoard({ board, lastMove, showLastMove }: { board: [PieceColour | null, PieceVariant | null][], lastMove: [number, number], showLastMove: boolean }) {
  const boardRef = useRef<HTMLDivElement | null>(null)
  const rect = useRef<Rect | null>(null)
  const [boardWidth, setBoardWidth] = useState(boardRef.current?.getBoundingClientRect().height || 200)
  const squareWidth = boardWidth / 8

  useEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const boundingRect = entry.target.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.height)
      }
    })
    if (boardRef.current) {
      resizeObserver.observe(boardRef.current)
    }

    window.addEventListener('scroll', () => {
      if (boardRef.current) {
        const boundingRect = boardRef.current.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.height)
      }
    })

    window.addEventListener('resize', () => {
      if (boardRef.current) {
        const boundingRect = boardRef.current.getBoundingClientRect()
        rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
        setBoardWidth(boundingRect.height)
      }
    })

    return () => {
      resizeObserver.disconnect()

      window.removeEventListener('scroll', () => {
        if (boardRef.current) {
          const boundingRect = boardRef.current.getBoundingClientRect()
          rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
          setBoardWidth(boundingRect.height)
        }
      })

      window.removeEventListener('resize', () => {
        if (boardRef.current) {
          const boundingRect = boardRef.current.getBoundingClientRect()
          rect.current = getRect(boundingRect.top, boundingRect.left, boundingRect.width, boundingRect.height)
          setBoardWidth(boundingRect.height)
        }
      })

    }
  }, [boardRef])

  // style={{width: `${squareWidth * 8}px`, height: `${squareWidth * 8}px`, backgroundSize: `${squareWidth * 8}px`}} 
  
  return (
      <div className='chessboard' style={{width: "35vh", height: "35vh", backgroundSize: `${squareWidth * 8}px`}} ref={boardRef}>
        <LastMoveComponent flip={false} squareWidth={squareWidth} lastMove={lastMove} showLastMove={showLastMove}/>
        {board.map((square, idx) => {
          const [colour, variant] = square
          return (
            <PiecesComponent flip={false} squareWidth={squareWidth} colour={colour} variant={variant} index={idx} onDragEndCallback={() => {}}/>
          )
        })}
      </div>
  )
}