// Traducción al español de content/ru.ts. La versión rusa es la jurídicamente vinculante; esta es una traducción de cortesía.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "En resumen",
      body: [
        "KISY es una aplicación de mensajería. Para que funcione, guardamos tu cuenta, tus conversaciones y los archivos que envías. No vendemos estos datos, no mostramos publicidad y no los cedemos a terceros, salvo en los casos descritos más abajo.",
        "Las conversaciones privadas uno a uno se cifran en los dispositivos: el servidor las guarda de una forma que él mismo no puede leer. Los mensajes de los grupos y las publicaciones de las comunidades se guardan sin cifrar: el administrador del servidor puede verlos.",
        "Puedes eliminar tu cuenta en cualquier momento: Perfil → «Eliminar cuenta», o en la página de eliminación de la cuenta, sin entrar en la app.",
      ],
    },
    {
      title: "Quién es responsable de los datos",
      body: [
        "El servicio funciona como un proyecto privado. El responsable del tratamiento de los datos es el propietario de la instalación de KISY en la que te registraste; es también quien responde a las preguntas sobre tus datos en la dirección indicada al final de este documento.",
        "La app es gratuita y se distribuye «tal cual», sin garantías de disponibilidad ni de conservación de los datos (consulta la sección «Responsabilidad»).",
      ],
    },
    {
      title: "Qué datos recopilamos",
      body: [
        "— Cuenta: nombre de usuario, nombre visible, hash de la contraseña (la contraseña en sí no se guarda), fecha de registro, nivel de acceso y foto de perfil, si la subiste.",
        "— Contenido: mensajes, archivos, mensajes de voz, notas, publicaciones en comunidades, reacciones, calendario y tareas; todo lo que creas en la app.",
        "— Datos técnicos: horas de inicio de sesión, una huella irreversible (hash) de la dirección IP, nombre del dispositivo y del navegador, identificadores de notificaciones push y el registro de acciones de los administradores.",
        "— Llamadas: que la llamada tuvo lugar, su hora y su duración. El contenido de las llamadas no se graba.",
        "— Consentimiento: cuándo aceptaste esta política y las normas de la comunidad, qué versiones de ellas y una huella irreversible de tu dirección IP en ese momento. Es la prueba del consentimiento, por eso se conserva incluso después de eliminar la cuenta, igual que el registro de seguridad.",
        "No pedimos ni guardamos tu número de teléfono, tu dirección, tus datos de pago ni tu ubicación exacta.",
      ],
    },
    {
      title: "Para qué los necesitamos",
      body: [
        "— Para entregar los mensajes y mostrar las conversaciones en tus dispositivos.",
        "— Para proteger el servicio: limitar el spam y los intentos de adivinar contraseñas y detectar abusos (para eso necesitamos el hash de la dirección IP y no la dirección en sí).",
        "— Para enviarte notificaciones, si las has activado.",
        "— Para cumplir los requisitos legales cuando nos sean aplicables.",
        "No usamos tus datos para publicidad, elaboración de perfiles ni entrenamiento de modelos.",
      ],
    },
    {
      title: "Quién tiene acceso a los datos",
      body: [
        "— Otros usuarios: exactamente las personas a las que escribes y los miembros de los grupos y comunidades donde escribes.",
        "— Los proveedores de infraestructura sin los que el servicio no funciona: alojamiento de la app y de la base de datos, almacenamiento de archivos, servicio de entrega de notificaciones push (Google Firebase) y protección contra bots (Cloudflare Turnstile). Tratan los datos por encargo nuestro y no los usan para sus propios fines.",
        "— El administrador del servidor, en la medida descrita en la sección «Qué ve el administrador».",
        "No vendemos datos ni los cedemos a redes publicitarias.",
      ],
    },
    {
      title: "Qué ve el administrador",
      body: [
        "Con franqueza sobre los límites del cifrado. El administrador del servidor no puede leer las conversaciones privadas uno a uno: están cifradas con claves que permanecen en los dispositivos.",
        "El administrador puede ver: los mensajes de los chats de grupo y las publicaciones de las comunidades, los nombres y nombres de usuario, la existencia de conversaciones y su hora (quién habla con quién y cuándo), los archivos subidos y el registro de acciones. Cuando se denuncia un mensaje privado, solo vemos que existe la denuncia: el texto sigue cifrado.",
        "Si necesitas que tus conversaciones sean totalmente confidenciales, usa los chats privados, no los grupos.",
      ],
    },
    {
      title: "Cuánto tiempo conservamos los datos",
      body: [
        "Los mensajes y archivos se conservan hasta que los elimines o hasta que elimines tu cuenta. Los mensajes temporales se eliminan según el temporizador que hayas configurado.",
        "El registro de acciones y los registros de inicio de sesión se conservan hasta un año: son necesarios para investigar intrusiones.",
        "Las copias de seguridad de la base de datos se conservan cifradas hasta 30 días. Los datos eliminados desaparecen de las copias a medida que estas caducan.",
      ],
    },
    {
      title: "Eliminación de la cuenta",
      body: [
        "Puedes eliminar tu cuenta en la app: Perfil → «Eliminar cuenta», con confirmación mediante tu contraseña. Lo mismo puedes hacer en la página de eliminación de la cuenta, sin instalar la app.",
        "La eliminación es inmediata y no se puede deshacer. Se eliminan: la contraseña y todas las sesiones, las claves de cifrado, los tokens push, tus archivos y notas, los ajustes, las reacciones y los votos, y el texto de tus mensajes privados, también para la otra persona.",
        "Se conservan: tus mensajes en los chats de grupo y tus publicaciones en las comunidades (son conversaciones de otras personas y la sección pública de Novedades), ya sin tu nombre, a nombre de «Cuenta eliminada»; y el registro de seguridad, que no se puede reescribir a posteriori.",
        "Los grupos y comunidades que administrabas pasan a la siguiente persona a cargo; si no la hay, se eliminan junto con la cuenta. Tu nombre de usuario no pasará a nadie más.",
      ],
    },
    {
      title: "Tus derechos",
      body: [
        "Puedes obtener una copia de tus datos, rectificarlos, eliminar tu cuenta u oponerte al tratamiento: escríbenos a la dirección que figura al final de este documento. Respondemos en un plazo razonable, normalmente dentro de los 30 días.",
        "Si estás en la UE o en el Reino Unido, tienes derecho a presentar una reclamación ante la autoridad de control de tu país.",
      ],
    },
    {
      title: "La seguridad y sus límites",
      body: [
        "Lo que hacemos: las contraseñas se guardan solo como hashes (Argon2id), las conversaciones privadas se cifran en los dispositivos, el tráfico va por TLS, las acciones de los administradores quedan registradas, las copias de seguridad se cifran y hay protección contra los intentos de adivinar contraseñas y los registros de spam.",
        "Lo que no prometemos. Ningún servicio está protegido por completo. Riesgos que debes conocer de antemano:",
        "— Una intrusión en el servidor o en la infraestructura de un proveedor puede dejar expuesto todo, salvo el contenido de las conversaciones privadas.",
        "— El acceso a tu dispositivo (robo, un programa malicioso, que otra persona lo tenga en sus manos) deja expuestas también las conversaciones privadas: las claves se guardan en el dispositivo.",
        "— La otra persona puede guardar, reenviar o fotografiar lo que le enviaste. Técnicamente no se puede impedir.",
        "— Perder el dispositivo o reinstalar la app puede suponer perder el historial de las conversaciones privadas: las claves de cifrado no salen del dispositivo y no podemos recuperarlas por ti.",
        "— Un fallo del alojamiento, un error en el programa o el agotamiento del plan gratuito pueden dejar el servicio sin disponibilidad y provocar la pérdida de datos. Haz tú mismo copias de seguridad de lo importante.",
        "— Las notificaciones push pasan por Google y nosotros no las ciframos en su camino hasta el dispositivo; pueden incluir el nombre del remitente y el comienzo del mensaje si no has desactivado la vista previa.",
      ],
    },
    {
      title: "Menores",
      body: [
        "El servicio no está destinado a menores de 13 años (en la UE, a menores de 16 años o de la edad que establezca tu país). Si te enteras de que un menor usa el servicio sin el consentimiento de sus padres, escríbenos y eliminaremos la cuenta.",
      ],
    },
    {
      title: "Responsabilidad",
      body: [
        "El servicio se ofrece «tal cual» y «según disponibilidad», sin garantías de ningún tipo: de disponibilidad, de conservación de los datos, de idoneidad para un fin concreto ni de ausencia de errores.",
        "En la máxima medida permitida por la legislación aplicable, el propietario del servicio no se hace responsable de: la pérdida o el deterioro de datos y conversaciones, la falta de disponibilidad del servicio, la imposibilidad de recuperar las claves de cifrado y el historial, el contenido creado por los usuarios, las acciones de otros usuarios y de terceros, ni de los daños indirectos, el lucro cesante y cualquier consecuencia del uso o de la imposibilidad de usar el servicio.",
        "El contenido de las conversaciones lo crean los usuarios. El propietario del servicio no lo revisa de antemano ni responde de él; si se incumplen las normas, la cuenta puede ser restringida o eliminada.",
        "Las salvedades anteriores no excluyen lo que por ley no se puede excluir: la responsabilidad por actos dolosos y negligencia grave y por daños a la vida y la salud, ni los derechos de los consumidores y las obligaciones del responsable del tratamiento de datos personales en tu país. Donde tal limitación no esté permitida, se aplica en la mínima medida permitida por la ley.",
      ],
    },
    {
      title: "Cambios",
      body: [
        "Podemos modificar esta política. Los cambios importantes se muestran en la app. La fecha del último cambio figura al principio de la página.",
      ],
    },
    {
      title: "Contacto",
      body: [
        "Preguntas sobre los datos, solicitudes de eliminación y reclamaciones: kisyandco@gmail.com. Respondemos en un plazo razonable, normalmente dentro de los 30 días.",
      ],
    },
  ],
  deletion: [
    {
      title: "Cómo eliminar tu cuenta desde la app",
      body: [
        "1. Abre KISY e inicia sesión en tu cuenta.",
        "2. Perfil (el icono de abajo a la derecha) → «Eliminar cuenta».",
        "3. Escribe tu contraseña y la palabra ELIMINAR.",
        "4. Listo: la cuenta se elimina al instante y no se puede deshacer.",
      ],
    },
    {
      title: "Qué se elimina",
      body: [
        "— La contraseña, todas las sesiones activas y los dispositivos.",
        "— Las claves de cifrado y los tokens push.",
        "— Tus archivos, notas, ajustes, reacciones y votos.",
        "— El texto de tus mensajes privados, también para la otra persona.",
      ],
    },
    {
      title: "Qué se conserva",
      body: [
        "— Tus mensajes en los chats de grupo y tus publicaciones en las comunidades, sin tu nombre, a nombre de «Cuenta eliminada». Son conversaciones de otras personas y la sección pública de Novedades, y no podemos borrarlas en nombre de los demás.",
        "— El registro de seguridad (quién inició sesión y cuándo, acciones de los administradores), hasta un año, sin el contenido de las conversaciones.",
        "— El registro de que aceptaste la política y las normas de la comunidad: cuándo y qué versiones de ellas. Sin él no se puede demostrar que hubo consentimiento.",
        "— Las copias de seguridad cifradas de la base de datos, hasta 30 días; después desaparecen.",
      ],
    },
    {
      title: "Si no puedes iniciar sesión",
      body: [
        "Escribe a kisyandco@gmail.com indicando el nombre de usuario que quieres eliminar. Eliminaremos la cuenta tras comprobar que es tuya.",
      ],
    },
  ],
  rules: [
    {
      title: "En resumen",
      body: [
        "KISY es un lugar para conversar, para grupos compartidos y para comunidades. Las normas siguientes se aplican allí donde escribas, subas o muestres algo a los demás: en los chats privados y de grupo, en las comunidades y en Novedades, en los nombres y descripciones de los grupos, y en el nombre y la foto de tu perfil.",
        "La norma principal: no hagas a los demás nada por lo que, en la vida real, tendrías que responder ante la ley o ante otras personas. Si tienes dudas, no lo publiques.",
      ],
    },
    {
      title: "Qué está prohibido",
      body: [
        "— El abuso sexual infantil y cualquier material que sexualice a menores. Aquí no hay advertencias: la cuenta se bloquea de inmediato y la información se entrega a las fuerzas de seguridad.",
        "— El contenido sexual explícito y la pornografía. El servicio está pensado para usuarios a partir de 13 años.",
        "— Las amenazas, los llamamientos a la violencia, la apología de la violencia y del terrorismo, y la captación para organizaciones extremistas.",
        "— El acoso y la persecución: insultos, ataques sistemáticos contra una persona, azuzar a otros contra alguien, mensajes repetidos a quien te ha bloqueado o te ha pedido que pares.",
        "— La incitación al odio y la humillación de personas por su nacionalidad, raza, religión, sexo, orientación sexual, discapacidad, edad u origen.",
        "— Publicar datos personales de otras personas sin su consentimiento: direcciones, teléfonos, documentos, fotos de su vida privada, conversaciones.",
        "— Incitar a la autolesión y al suicidio, dar instrucciones para ello, idealizar los trastornos de la conducta alimentaria.",
        "— El spam y la manipulación de la interacción: mensajes idénticos masivos, publicidad no solicitada, granjas de cuentas, inflar artificialmente reacciones y votos.",
        "— El fraude y el engaño: phishing, sacar dinero o contraseñas con engaños, sorteos falsos, archivos y enlaces maliciosos.",
        "— Hacerse pasar por otra persona u organización, incluido usar un nombre y una foto de perfil que induzcan a error.",
        "— Vender y anunciar productos prohibidos: drogas, armas, documentos falsificados, objetos robados.",
        "— Vulnerar los derechos de otros: publicar obras, fotos y materiales ajenos sin tener derecho a ello.",
        "— Eludir restricciones: crear una cuenta nueva para seguir haciendo aquello por lo que se restringió o eliminó la anterior.",
      ],
    },
    {
      title: "Grupos y comunidades",
      body: [
        "Quien crea un grupo o una comunidad y los editores que nombre son responsables del orden dentro de ellos y pueden eliminar publicaciones. Las normas internas de una comunidad no pueden permitir lo que estas normas prohíben.",
        "Una comunidad cerrada no es un lugar donde las normas no se aplican: ser cerrada protege a los miembros de las miradas ajenas, no a las infracciones de la moderación.",
      ],
    },
    {
      title: "Conversaciones privadas",
      body: [
        "Los chats privados están cifrados en los dispositivos y no podemos leerlos, tampoco cuando se denuncian: de un mensaje de un chat privado solo vemos que existe la denuncia. Por eso, aquí la principal protección está en tus manos: puedes bloquear a la otra persona en cualquier momento, y las decisiones sobre una cuenta se toman en función del conjunto de las denuncias.",
        "El cifrado no convierte lo prohibido en permitido. Si la otra persona nos muestra ella misma una infracción (por ejemplo, con una captura de pantalla al contactarnos), tenemos derecho a tomar medidas conforme a estas normas.",
      ],
    },
    {
      title: "Cómo protegerte y denunciar",
      body: [
        "— Bloquear. El botón está en la cabecera del chat privado con esa persona. El bloqueo es unilateral y silencioso: la otra persona no se enterará, no podrá enviarte mensajes privados ni llamarte, y sus publicaciones en las comunidades desaparecerán de tus Novedades. Puedes quitar el bloqueo en Perfil → «Bloqueados».",
        "— Puedes denunciar un mensaje (desde su menú), una publicación de una comunidad, a una persona (en la cabecera del chat privado o en la lista de miembros del grupo) y la propia comunidad o grupo. La persona a la que denuncies no se enterará.",
        "— Solo puedes denunciar lo que tú mismo ves.",
        "— Una publicación de una comunidad denunciada por cinco personas distintas se oculta de Novedades hasta que se revise. Las denuncias de cuentas creadas en las últimas horas llegan a la administración, pero no cuentan para esas cinco; de lo contrario, se podría ocultar una publicación con un puñado de cuentas recién creadas.",
        "— Las denuncias las revisa la administración del servicio. Si te amenazan o estás en peligro, acude primero a la policía: no somos un servicio de emergencias.",
      ],
    },
    {
      title: "Consecuencias de las infracciones",
      body: [
        "Según la gravedad y la reincidencia:",
        "— eliminación de la publicación de la comunidad, por parte de sus editores o de la administración;",
        "— para un grupo o una comunidad: advertencia, exclusión de Novedades, eliminación (la tercera advertencia vigente elimina la comunidad);",
        "— para una cuenta: bloqueo, tras el cual ya no se puede iniciar sesión en ella.",
        "Ante infracciones graves (amenazas contra la vida, abuso sexual infantil, terrorismo), la cuenta se bloquea sin advertencias y la información puede entregarse a las fuerzas de seguridad conforme al procedimiento establecido por la ley.",
        "Una cuenta creada sin invitación funciona en modo restringido durante las primeras horas tras el registro: es una protección contra el spam, no un castigo.",
      ],
    },
    {
      title: "Si no estás de acuerdo con una decisión",
      body: [
        "Escribe a kisyandco@gmail.com indicando tu nombre de usuario, qué ocurrió y por qué crees que la decisión es errónea. La revisaremos y te responderemos, normalmente dentro de los 30 días.",
      ],
    },
    {
      title: "Cambios en las normas",
      body: [
        "Las normas pueden cambiar. Si hay cambios importantes, la app te pedirá que aceptes la nueva versión la próxima vez que inicies sesión; sin hacerlo no se puede usar el servicio. La fecha del último cambio figura al principio de la página.",
      ],
    },
  ],
};
