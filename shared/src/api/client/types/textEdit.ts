import { Position, Range } from '@sourcegraph/extension-api-classes'
import * as sourcegraph from 'sourcegraph'

export class TextEdit implements sourcegraph.TextEdit {
    public static isTextEdit(thing: any): thing is TextEdit {
        if (thing instanceof TextEdit) {
            return true
        }
        if (!thing) {
            return false
        }
        // tslint:disable-next-line: strict-type-predicates
        return Range.isRange(thing as TextEdit) && typeof (thing as TextEdit).newText === 'string'
    }

    public static replace(range: Range, newText: string): TextEdit {
        return new TextEdit(range, newText)
    }

    public static insert(position: Position, newText: string): TextEdit {
        return TextEdit.replace(new Range(position, position), newText)
    }

    public static delete(range: Range): TextEdit {
        return TextEdit.replace(range, '')
    }

    constructor(public readonly range: Range, public readonly newText: string) {}

    public toJSON(): any {
        return {
            range: this.range,
            newText: this.newText,
        }
    }
}
