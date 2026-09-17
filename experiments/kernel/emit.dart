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
    'int' => core.intNonNullableRawType,
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

  Expression expression(Map<String, dynamic> n) {
    switch (n['op']) {
      case 'int':
        return IntLiteral(int.parse(n['value']));
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
                if (a['op'] == 'string')
                  expression(a)
                else
                  StringConcatenation([expression(a)]),
            ], typeArgument: core.stringNonNullableRawType),
          ]),
        );
      case 'binary':
        final op = n['operator'] == '!=' ? '==' : n['operator'];
        final target = operatorTarget(op);
        final call = InstanceInvocation(
          InstanceAccessKind.Instance,
          expression(n['left']),
          Name(op),
          Arguments([expression(n['right'])]),
          interfaceTarget: target,
          functionType: target.function.computeFunctionType(
            Nullability.nonNullable,
          ),
          resultType: type(n['type']),
        );
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
    final uri = File(program['source']).absolute.uri;
    final source = File.fromUri(uri).readAsBytesSync();
    final lines = <int>[0];
    for (var i = 0; i < source.length; i++) {
      if (source[i] == 10) lines.add(i + 1);
    }
    component.uriToSource[uri] = Source(lines, source, uri, uri);
    final library = Library(
      uri,
      fileUri: uri,
      dependencies: [
        LibraryDependency.import(core.coreLibrary),
        LibraryDependency.import(println.enclosingLibrary),
      ],
    )..setLanguageVersion(Version(3, 13));
    for (final f in program['functions']) {
      final params = <PositionalParameter>[
        for (final p in f['params'])
          PositionalParameter(cosmeticName: p['name'], type: type(p['type'])),
      ];
      final proc = Procedure(
        Name(f['name']),
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
    for (final f in program['functions']) {
      final fn = functions[f['name']]!.function;
      variables = {for (final p in fn.positionalParameters) p.cosmeticName!: p};
      context = VariableContext(
        captureKind: CaptureKind.notCaptured,
        variables: [...fn.positionalParameters],
      );
      fn.scope = Scope(contexts: [context]);
      fn.body = statement(f['body'])..parent = fn;
    }
    component.libraries.add(library..parent = component);
    component.setMainMethodAndMode(functions['main']!.reference, true);
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
