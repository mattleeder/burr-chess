export enum PieceColour {
    White,
    Black,
    Spectator,
}

export enum PieceVariant {
    Pawn,
    Knight,
    Bishop,
    Rook,
    Queen,
    King,
}

const charToPiece = new Map<string, [PieceColour, PieceVariant]>()

// White
charToPiece.set('P', [PieceColour.White, PieceVariant.Pawn])
charToPiece.set('N', [PieceColour.White, PieceVariant.Knight])
charToPiece.set('B', [PieceColour.White, PieceVariant.Bishop])
charToPiece.set('R', [PieceColour.White, PieceVariant.Rook])
charToPiece.set('Q', [PieceColour.White, PieceVariant.Queen])
charToPiece.set('K', [PieceColour.White, PieceVariant.King])

// Black
charToPiece.set('p', [PieceColour.Black, PieceVariant.Pawn])
charToPiece.set('n', [PieceColour.Black, PieceVariant.Knight])
charToPiece.set('b', [PieceColour.Black, PieceVariant.Bishop])
charToPiece.set('r', [PieceColour.Black, PieceVariant.Rook])
charToPiece.set('q', [PieceColour.Black, PieceVariant.Queen])
charToPiece.set('k', [PieceColour.Black, PieceVariant.King])

const variantToChar = new Map<PieceVariant, string>()

variantToChar.set(PieceVariant.Pawn, 'p')
variantToChar.set(PieceVariant.Knight, 'n')
variantToChar.set(PieceVariant.Bishop, 'b')
variantToChar.set(PieceVariant.Rook, 'r')
variantToChar.set(PieceVariant.Queen, 'q')
variantToChar.set(PieceVariant.King, 'k')

const fileToNumber = new Map<string, number>()
fileToNumber.set('a', 0)
fileToNumber.set('b', 1)
fileToNumber.set('c', 2)
fileToNumber.set('d', 3)
fileToNumber.set('e', 4)
fileToNumber.set('f', 5)
fileToNumber.set('g', 6)
fileToNumber.set('h', 7)

interface gameState {
    fen: string,
    board: [PieceColour | null, PieceVariant | null][],
    activeColour: PieceColour,
    blackCanKingSideCastle: boolean,
    blackCanQueenSideCastle: boolean,
    whiteCanKingSideCastle: boolean,
    whiteCanQueenSideCastle: boolean,
    enPassantSquare: number | null,
}


export function parseGameStateFromFEN(fen: string): gameState {
  // Board, turn, castling, enpassant, halfmove, fullmove
  const args = fen.split(" ")
  const game: gameState = {
    fen: fen,
    board: [],
    activeColour: PieceColour.White,
    blackCanKingSideCastle: false,
    blackCanQueenSideCastle: false,
    whiteCanKingSideCastle: false,
    whiteCanQueenSideCastle: false,
    enPassantSquare: null
  }

  for (const char of args[0]) {
    if (char == "/") {
      continue
    }

    // Digit
    if (char >= '1' && char <= '8') {
      for (let i = 0; i < parseInt(char); i++) {
        game.board.push([null, null])
      }
      continue
    }
        
    const piece = charToPiece.get(char)
    if (piece) {
      game.board.push(piece)
    }

  }

  if (args[1] == "b") {
    game.activeColour = PieceColour.Black
  }

  for (const char of args[2]) {
    switch (char) {
    case "K":
      game.whiteCanKingSideCastle = true
      break
    case "Q":
      game.whiteCanQueenSideCastle = true
      break
    case "k":
      game.blackCanKingSideCastle = true
      break
    case "q":
      game.blackCanQueenSideCastle = true
      break
    }
  }

  if (args[3].length > 1) {
    const file = fileToNumber.get(args[3][0])
    const rank = parseInt(args[3][1])
    if (file !== undefined) {
      game.enPassantSquare = file + (8 - rank) * 8
    }
  }

  return game
}

export function gameStateToFEN(currentState: gameState): string {
  const fen: string[] = []

  let rowCount = 0
  let emptyCount = 0

  for (const [colour, variant] of currentState.board) {
        
    rowCount += 1

    if (variant === null || colour === null) {
      emptyCount += 1
    } else {
            
      if (emptyCount > 0) {
        fen.push(emptyCount.toString())
        emptyCount = 0
      }
            
      let char = variantToChar.get(variant)
            
      if (char == undefined) {
        throw new Error("Unable to convert variant to char")
      }
            
      if (colour == PieceColour.White) {
        char = char?.toUpperCase()
      }
            
      fen.push(char)
    }

    if (rowCount >= 8) {
      if (emptyCount > 0) {
        fen.push(emptyCount.toString())
        emptyCount = 0
      }
      fen.push("/")
      rowCount = 0
    }
  }

  // Active colour
  fen.push(currentState.activeColour === PieceColour.Black ? " b " : " w ")

  // Castling rights
  let castling = ""
  if (currentState.whiteCanKingSideCastle) castling += "K"
  if (currentState.whiteCanQueenSideCastle) castling += "Q"
  if (currentState.blackCanKingSideCastle) castling += "k"
  if (currentState.blackCanQueenSideCastle) castling += "q"
  fen.push(castling || "-")

  // En passant square
  if (currentState.enPassantSquare !== null) {
    const file = currentState.enPassantSquare % 8
    const rank = 8 - Math.floor(currentState.enPassantSquare / 8)
    fen.push(" " + String.fromCharCode("a".charCodeAt(0) + file) + rank)
  } else {
    fen.push(" -")
  }

  fen.push(" 0 1")

  return fen.join("")
}