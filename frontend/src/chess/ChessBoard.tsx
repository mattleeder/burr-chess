import React, { useContext, useEffect, useReducer, useRef, useState } from "react";
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

interface ClickState {
  waiting: boolean
  selectedPiece: number | null
  moves: number[]
  captures: number[]
  promotionNextMove: boolean
  promotionActive: boolean
  promotionSquare: number
}

type ClickStateAction =
  | { type: 'clear' }
  | { type: 'setWaiting'; waiting: boolean }
  | { type: 'showMoves'; piece: number; moves: number[]; captures: number[]; promotionNextMove: boolean }
  | { type: 'startPromotion'; square: number }

const initialClickState: ClickState = {
  waiting: false,
  selectedPiece: null,
  moves: [],
  captures: [],
  promotionNextMove: false,
  promotionActive: false,
  promotionSquare: 0,
}

function clickStateReducer(state: ClickState, action: ClickStateAction): ClickState {
  switch (action.type) {
  case 'clear':
    return { ...initialClickState, waiting: state.waiting }
  case 'setWaiting':
    return { ...state, waiting: action.waiting }
  case 'showMoves':
    return { ...state, selectedPiece: action.piece, moves: action.moves, captures: action.captures, promotionNextMove: action.promotionNextMove }
  case 'startPromotion':
    return { ...state, promotionActive: true, promotionSquare: action.square, promotionNextMove: false }
  }
}

function getRect(top: number, left: number, width: number, height: number): Rect {
  return { top, left, width, height }
}

function useBoardSize(boardRef: React.RefObject<HTMLDivElement | null>, initialWidth: number = 200) {
  const rect = useRef<Rect | null>(null)
  const [boardWidth, setBoardWidth] = useState(initialWidth)

  useEffect(() => {
    const update = () => {
      if (!boardRef.current) return
      const b = boardRef.current.getBoundingClientRect()
      rect.current = getRect(b.top, b.left, b.width, b.height)
      setBoardWidth(b.width)
    }

    const resizeObserver = new ResizeObserver(update)
    if (boardRef.current) {
      resizeObserver.observe(boardRef.current)
    }

    window.addEventListener('scroll', update)
    window.addEventListener('resize', update)

    return () => {
      resizeObserver.disconnect()
      window.removeEventListener('scroll', update)
      window.removeEventListener('resize', update)
    }
  }, [boardRef])

  return { rect, boardWidth }
}

function getSquareIdxFromClick(x: number, y: number, rect?: React.RefObject<Rect | null>) {
  if (rect === undefined || rect === null || rect.current === null) {
    throw new Error("Bounding rect for board is not defined")
  }

  const squareWidth = rect.current.width / 8
  const boardXPosition = Math.floor((x - rect.current.left) / squareWidth)
  const boardYPosition = Math.floor((y - rect.current.top) / squareWidth)
  return boardYPosition * 8 + boardXPosition
}

async function clickHandler(
  position: number,
  game: gameContext,
  state: ClickState,
  dispatch: React.Dispatch<ClickStateAction>,
) {
  if (game.flip) {
    position = 63 - position
  }

  if (state.waiting) return

  dispatch({ type: 'setWaiting', waiting: true })

  let clickAction = ClickAction.clear
  if (game.matchData.activeMove !== game.matchData.stateHistory.length - 1) {
    dispatch({ type: 'setWaiting', waiting: false })
    return
  } else if (game.matchData.gameOverStatus !== 0) {
    dispatch({ type: 'setWaiting', waiting: false })
    return
  } else if (state.promotionActive && [0, 8, 16, 24].includes(Math.abs(position - state.promotionSquare))) {
    clickAction = ClickAction.choosePromotion
  } else if ([...state.moves, ...state.captures].includes(position)) {
    clickAction = ClickAction.makeMove
  } else if (game.matchData.activeState.board[position][0] === game.playerColour && position !== state.selectedPiece) {
    clickAction = ClickAction.showMoves
  }

  switch (clickAction) {
  case ClickAction.clear:
    dispatch({ type: 'clear' })
    break

  case ClickAction.makeMove:
  case ClickAction.choosePromotion: {
    if (state.selectedPiece === null) {
      throw new Error("Posting move with no piece")
    }

    if (state.promotionNextMove) {
      dispatch({ type: 'startPromotion', square: position })
      break
    }

    let promotionString = ""
    if (clickAction === ClickAction.choosePromotion) {
      const promotionIndex = [0, 8, 16, 24].indexOf(Math.abs(position - state.promotionSquare))
      promotionString = "qnrb"[promotionIndex]
      position = state.promotionSquare
    }
    wsPostMove(position, state.selectedPiece, promotionString, game)
    dispatch({ type: 'clear' })
    break
  }

  case ClickAction.showMoves: {
    const data = await fetchPossibleMoves(position, game)
    dispatch({
      type: 'showMoves',
      piece: position,
      moves: data?.moves ?? [],
      captures: data?.captures ?? [],
      promotionNextMove: data?.triggerPromotion ?? false,
    })
    break
  }
  }

  dispatch({ type: 'setWaiting', waiting: false })
}

async function fetchPossibleMoves(position: number, game: gameContext) {
  try {
    const mostRecentMove = game.matchData.stateHistory.at(-1)
    if (!mostRecentMove) return {}

    const response = await fetch(API.fetchMoves, {
      method: "POST",
      body: JSON.stringify({
        fen: mostRecentMove["FEN"],
        piece: position,
      })
    })

    if (!response.ok) {
      throw new Error(`Response status: ${response.status}`)
    }

    return await response.json()
  } catch (error: unknown) {
    if (error instanceof Error) {
      console.error(error.message)
    } else {
      console.error(error)
    }
    return {}
  }
}

function wsPostMove(position: number, piece: number, promotion: string, game: gameContext) {
  game.webSocket.current?.send(JSON.stringify({
    messageType: "postMove",
    body: {
      piece,
      move: position,
      promotionString: promotion,
    }
  }))
}

function getRowAndColFromBoardIndex(idx: number, flip: boolean): [number, number] {
  const row = Math.floor(idx / 8)
  const col = idx % 8
  if (flip) {
    return [Math.abs(7 - row), Math.abs(7 - col)]
  }
  return [row, col]
}

function PiecesComponent({ flip, squareWidth, onDragEndCallback, rect, colour, variant, index }: { flip: boolean, squareWidth: number, onDragEndCallback: (startIdx: number, endIdx: number) => void, rect?: React.RefObject<Rect | null>, colour: PieceColour | null, variant: PieceVariant | null, index: number }) {
  if (colour === null || variant === null) {
    return <></>
  }

  const [row, col] = getRowAndColFromBoardIndex(index, flip)

  return (
    <div
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
  return moves.map((move, idx) => {
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
  })
}

function CapturesComponent({ captures, flip, squareWidth }: { captures: number[], flip: boolean, squareWidth: number }) {
  return captures.map((move, idx) => {
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
  })
}

function LastMoveComponent({ flip, squareWidth, lastMove, showLastMove }: { flip: boolean, squareWidth: number, lastMove: [number, number], showLastMove: boolean }) {
  return lastMove.map((move, idx) => {
    if (!showLastMove) return <React.Fragment key={idx} />
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
  })
}

const PROMOTION_PIECES = ['queen', 'knight', 'rook', 'bishop'] as const

function PromotionComponent({ promotionSquare, promotionActive, flip, squareWidth }: { promotionSquare: number, promotionActive: boolean, flip: boolean, squareWidth: number }) {
  if (!promotionActive) return <></>

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_row, col] = getRowAndColFromBoardIndex(promotionSquare, flip)
  const promotionColour = promotionSquare <= 7 ? colourToString.get(PieceColour.White) : colourToString.get(PieceColour.Black)

  return (
    <>
      {PROMOTION_PIECES.map((piece, i) => (
        <div
          key={piece}
          className={`${promotionColour}-${piece} promotion`}
          style={{
            transform: `translate(${col * squareWidth}px, ${i * squareWidth}px)`,
            width: `${squareWidth}px`,
            height: `${squareWidth}px`,
            backgroundSize: `${squareWidth}px`,
          }}
        />
      ))}
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
  const gameOverText = gameOverStatusCodes[game.matchData.gameOverStatus || 0]

  return <div style={{ transform: `translate(${0}px, ${squareWidth * 4}px)`, color: "black" }}>{gameOverText}</div>
}

export function ChessBoard({ resizeable, defaultWidth, chessboardContainerStyles, enableClicking }: { resizeable: boolean, defaultWidth: number, chessboardContainerStyles?: React.CSSProperties, enableClicking: boolean }) {
  const boardRef = useRef<HTMLDivElement | null>(null)
  const { rect, boardWidth } = useBoardSize(boardRef, defaultWidth)
  const [clickState, dispatch] = useReducer(clickStateReducer, initialClickState)

  const game = useContext(GameContext)
  if (!game) {
    throw new Error('ChessBoard must be used within a GameContext Provider')
  }

  useEffect(() => {
    dispatch({ type: 'clear' })
  }, [game.matchData.activeMove])

  const squareWidth = boardWidth / 8

  const chessboardContainerStyle = { ...chessboardContainerStyles }
  chessboardContainerStyle["width"] = `${boardWidth}px`
  chessboardContainerStyle["height"] = `${boardWidth}px`
  if (resizeable) {
    chessboardContainerStyle["resize"] = "both"
    chessboardContainerStyle["overflow"] = "auto"
  }

  const handleClick = (position: number) => {
    if (enableClicking) {
      clickHandler(position, game, clickState, dispatch)
    }
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
          handleClick(getSquareIdxFromClick(event.clientX, event.clientY, rect))
        }}
      >
        <LastMoveComponent flip={game.flip} squareWidth={squareWidth} lastMove={game.matchData.activeState.lastMove} showLastMove={game.matchData.activeMove != 0} />
        {game.matchData.activeState.board.map((square, idx) => {
          const [colour, variant] = square
          return (
            <PiecesComponent
              key={idx}
              flip={game.flip}
              squareWidth={squareWidth}
              rect={rect}
              colour={colour}
              variant={variant}
              index={idx}
              onDragEndCallback={(startIdx, endIdx) => {
                if (startIdx !== endIdx) {
                  handleClick(endIdx)
                }
              }}
            />
          )
        })}
        <MovesComponent moves={clickState.moves} flip={game.flip} squareWidth={squareWidth} />
        <CapturesComponent captures={clickState.captures} flip={game.flip} squareWidth={squareWidth} />
        <PromotionComponent promotionSquare={clickState.promotionSquare} promotionActive={clickState.promotionActive} flip={game.flip} squareWidth={squareWidth} />
        <GameOverComponent squareWidth={squareWidth} />
      </div>
    </div>
  )
}

export function FrozenChessBoard({ board, lastMove, showLastMove }: { board: [PieceColour | null, PieceVariant | null][], lastMove: [number, number], showLastMove: boolean }) {
  const boardRef = useRef<HTMLDivElement | null>(null)
  const { boardWidth } = useBoardSize(boardRef)
  const squareWidth = boardWidth / 8

  return (
    <div className='chessboard' style={{ width: "35vh", height: "35vh", backgroundSize: `${squareWidth * 8}px` }} ref={boardRef}>
      <LastMoveComponent flip={false} squareWidth={squareWidth} lastMove={lastMove} showLastMove={showLastMove} />
      {board.map((square, idx) => {
        const [colour, variant] = square
        return (
          <PiecesComponent
            key={idx}
            flip={false}
            squareWidth={squareWidth}
            colour={colour}
            variant={variant}
            index={idx}
            onDragEndCallback={() => {}}
          />
        )
      })}
    </div>
  )
}
