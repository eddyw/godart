import 'dart:convert';
import 'dart:io';

import 'package:kernel/ast.dart';
import 'package:kernel/binary/ast_from_binary.dart';
import 'package:kernel/binary/ast_to_binary.dart';
import 'package:kernel/core_types.dart';

class Emitter {
  final Component component;
  late final CoreTypes core = CoreTypes(component);
  late final Procedure println = component.libraries
      .singleWhere(
        (l) => l.importUri.toString() == 'package:goruntime/src/print.dart',
      )
      .procedures
      .singleWhere((p) => p.name.text == r'$println');
  final Map<String, Procedure> functions = {};
  Map<String, Variable> variables = {};
  late VariableContext context;

  Emitter(this.component);

  DartType type(String name) => switch (name) {
    'int' || 'uint64' => core.intNonNullableRawType,
    'bool' => core.boolNonNullableRawType,
    'string' => core.stringNonNullableRawType,
    'void' => const VoidType(),
    _ => throw UnsupportedError('type $name'),
  };

  Procedure operatorTarget(String name) {
    Class? cls = core.intClass;
    while (cls != null) {
      for (final p in cls.procedures) {
        if (p.name.text == name) return p;
      }
      cls = cls.superclass;
    }
    throw StateError('int operator $name');
  }

  Expression runtimeCall(String name, List<Expression> args) {
    final library = component.libraries.singleWhere(
      (l) =>
          l.importUri.toString() == 'package:goruntime/src/goint_native.dart',
      orElse: () => throw UnsupportedError(
        '64-bit operations require VM or Wasm runtime',
      ),
    );
    final target = library.procedures.singleWhere((p) => p.name.text == name);
    return StaticInvocation(target, Arguments(args));
  }

  Expression intCall(
    String op,
    Expression receiver,
    List<Expression> args,
    DartType result,
  ) {
    final target = operatorTarget(op);
    return InstanceInvocation(
      InstanceAccessKind.Instance,
      receiver,
      Name(op),
      Arguments(args),
      interfaceTarget: target,
      functionType: target.function.computeFunctionType(
        Nullability.nonNullable,
      ),
      resultType: result,
    );
  }

  Expression expression(Map<String, dynamic> n) {
    switch (n['op']) {
      case 'int':
        return IntLiteral(BigInt.parse(n['value']).toSigned(64).toInt());
      case 'convert':
        return expression(n['value']);
      case 'unary':
        return intCall(
          n['operator'] == '^' ? '~' : 'unary-',
          expression(n['value']),
          [],
          type(n['type']),
        );
      case 'bool':
        return BoolLiteral(n['value']);
      case 'string':
        return StringLiteral(n['value']);
      case 'get':
        return VariableGet(variables[n['name']]!);
      case 'call':
        return StaticInvocation(
          functions[n['name']]!,
          Arguments([for (final a in n['args']) expression(a)]),
        );
      case 'println':
        return StaticInvocation(
          println,
          Arguments([
            ListLiteral([
              for (final a in n['args'])
                if (a['type'] == 'uint64')
                  runtimeCall(r'$ustring', [expression(a)])
                else if (a['op'] == 'string')
                  expression(a)
                else
                  StringConcatenation([expression(a)]),
            ], typeArgument: core.stringNonNullableRawType),
          ]),
        );
      case 'binary':
        final op = n['operator'] == '!=' ? '==' : n['operator'];
        var left = expression(n['left']);
        var right = expression(n['right']);
        if (op == '<<' || op == '>>') {
          if (n['unsignedCount'] == true)
            right = runtimeCall(r'$ucount', [right]);
          return runtimeCall(
            op == '<<'
                ? r'$shl'
                : n['unsigned'] == true
                ? r'$ushr'
                : r'$shr',
            [left, right],
          );
        }
        if (n['unsigned'] == true && ['<', '>', '<=', '>='].contains(op)) {
          return runtimeCall(
            op == '<' || op == '>' ? r'$ult' : r'$ule',
            op == '>' || op == '>=' ? [right, left] : [left, right],
          );
        }
        final call = intCall(op, left, [right], type(n['type']));
        return n['operator'] == '!=' ? Not(call) : call;
      default:
        throw UnsupportedError('expression ${n['op']}');
    }
  }

  Statement statement(Map<String, dynamic> n) {
    switch (n['op']) {
      case 'block':
        final outer = context;
        final outerVariables = variables;
        variables = {...variables};
        context = VariableContext(
          captureKind: CaptureKind.notCaptured,
          variables: [],
        );
        final block = Block([for (final s in n['items']) statement(s)])
          ..scope = Scope(contexts: [context]);
        context = outer;
        variables = outerVariables;
        return block;
      case 'declare':
        final v = LocalVariable(
          name: n['name'],
          type: type(n['type']),
          initializer: expression(n['value']),
          hasDeclaredInitializer: true,
        );
        variables[n['name']] = v;
        context.addVariable(v);
        return VariableStatement(VariableDeclaration(v));
      case 'set':
        return ExpressionStatement(
          VariableSet(variables[n['name']]!, expression(n['value'])),
        );
      case 'expression':
        return ExpressionStatement(expression(n['value']));
      case 'return':
        return ReturnStatement(
          n['value'] == null ? null : expression(n['value']),
        );
      case 'if':
        return IfStatement(
          expression(n['condition']),
          statement(n['then']),
          statement(n['else']),
        );
      case 'for':
        final outer = context;
        final outerVariables = variables;
        variables = {...variables};
        context = VariableContext(
          captureKind: CaptureKind.notCaptured,
          variables: [],
        );
        final init = statement(n['init']);
        final cond = n['condition'] == null ? null : expression(n['condition']);
        final body = statement(n['body']);
        final post = statement(n['post']);
        final block = Block([
          init,
          ForStatement([], cond, [], Block([body, post])),
        ])..scope = Scope(contexts: [context]);
        context = outer;
        variables = outerVariables;
        return block;
      default:
        throw UnsupportedError('statement ${n['op']}');
    }
  }

  void emit(Map<String, dynamic> program) {
    final libraries = <String, Library>{};
    for (final p in program['packages']) {
      final uri = Uri.parse('gart-kernel:${p['path']}');
      final library = Library(uri, fileUri: uri)
        ..setLanguageVersion(Version(3, 13));
      libraries[p['path']] = library;
      component.libraries.add(library..parent = component);
      for (final f in p['functions']) {
        final uri = File(f['source']).absolute.uri;
        if (!component.uriToSource.containsKey(uri)) {
          final source = File.fromUri(uri).readAsBytesSync();
          final lines = <int>[0];
          for (var i = 0; i < source.length; i++) {
            if (source[i] == 10) lines.add(i + 1);
          }
          component.uriToSource[uri] = Source(lines, source, uri, uri);
        }
        final params = <PositionalParameter>[
          for (final param in f['params'])
            PositionalParameter(
              cosmeticName: param['name'],
              type: type(param['type']),
            ),
        ];
        final proc = Procedure(
          Name(f['localName'], library),
          ProcedureKind.Method,
          FunctionNode(
            null,
            positionalParameters: params,
            returnType: type(f['result']),
          ),
          isStatic: true,
          fileUri: uri,
        )..fileOffset = f['offset'];
        functions[f['name']] = proc;
        library.addProcedure(proc);
      }
    }
    for (final p in program['packages']) {
      final library = libraries[p['path']]!;
      library.addDependency(LibraryDependency.import(core.coreLibrary));
      library.addDependency(LibraryDependency.import(println.enclosingLibrary));
      for (final runtime in component.libraries.where(
        (l) =>
            l.importUri.toString() == 'package:goruntime/src/goint_native.dart',
      )) {
        library.addDependency(LibraryDependency.import(runtime));
      }
      for (final dep in p['imports']) {
        library.addDependency(LibraryDependency.import(libraries[dep]!));
      }
      for (final f in p['functions']) {
        final fn = functions[f['name']]!.function;
        variables = {
          for (final param in fn.positionalParameters)
            param.cosmeticName!: param,
        };
        context = VariableContext(
          captureKind: CaptureKind.notCaptured,
          variables: [...fn.positionalParameters],
        );
        fn.scope = Scope(contexts: [context]);
        fn.body = statement(f['body'])..parent = fn;
      }
    }
    component.setMainMethodAndMode(functions[program['main']]!.reference, true);
  }
}

Future<void> main(List<String> args) async {
  if (!Platform.version.startsWith('3.13.3 ')) {
    throw StateError('This prototype requires Dart 3.13.3.');
  }
  final component = Component();
  BinaryBuilder(File(args[0]).readAsBytesSync()).readComponent(component);
  Emitter(component).emit(jsonDecode(File(args[1]).readAsStringSync()));
  final sink = File(args[2]).openWrite();
  BinaryPrinter(sink).writeComponentFile(component);
  await sink.close();
}
