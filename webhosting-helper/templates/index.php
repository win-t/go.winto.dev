<?php

function die_bad_gateway($msg) {
	if (!headers_sent()) {
		header_remove();
		header("HTTP/1.1 502 Bad Gateway");
		echo "Bad Gateway";
		flush();
	}
	trigger_error($msg, E_USER_ERROR);
	die();
}

$unix_socket = '{{ .service_sock }}';

if(ini_get("enable_post_data_reading")) die_bad_gateway(__LINE__);

$hop_by_hop_header = [
	"connection" => true,
	"keep-alive" => true,
	"proxy-authenticate" => true,
	"proxy-authorization" => true,
	"proxy-connection" => true,
	"te" => true,
	"trailer" => true,
	"transfer-encoding" => true,
	"upgrade" => true,
];


$in = fopen("php://input", "r");
if (!$in) die_bad_gateway(__LINE__);
function read_in($line, $length) {
	global $in;
	$data = fread($in, $length);
	if ($data === false) die_bad_gateway($line);
	return $data;
}

$out = fopen("php://output", "w");
if (!$out) die_bad_gateway(__LINE__);

$ch = curl_init();
if (!$ch) die_bad_gateway(__LINE__);

function ch_setopt($line, $key, $val) {
	global $ch;
	if (!curl_setopt($ch, $key, $val)) die_bad_gateway($line);
}

ch_setopt(__LINE__, CURLOPT_UNIX_SOCKET_PATH, $unix_socket);
ch_setopt(__LINE__, CURLOPT_CUSTOMREQUEST, $_SERVER["REQUEST_METHOD"]);
ch_setopt(__LINE__, CURLOPT_URL, "http://unix_socket" . $_SERVER["REQUEST_URI"]);

if (!isset($_SERVER["HTTP_CONNECTION"])) $remove_request_header = $hop_by_hop_header;
else {
	$remove_request_header = array_merge([], $hop_by_hop_header);
	foreach (explode(",", $_SERVER["HTTP_CONNECTION"]) as $key) {
		$key = strtolower(trim($key));
		if (strlen($key) > 0) $remove_request_header[$key] = true;
	}
}
$request_header = [];
foreach (getallheaders() as $key => $value) {
	if (isset($remove_request_header[strtolower($key)])) continue;
	$request_header[] = $key . ": " . $value;
}

// read 1 byte to check if the request has body
$preread = read_in(__LINE__, 1);
if (strlen($preread) == 0) ch_setopt(__LINE__, CURLOPT_HTTPHEADER, $request_header);
else {
	$request_header[] = "Expect:"; // remove implicit Expect: 100-continue header
	ch_setopt(__LINE__, CURLOPT_HTTPHEADER, $request_header);
	ch_setopt(__LINE__, CURLOPT_UPLOAD, true);
	if (isset($_SERVER["CONTENT_LENGTH"])) ch_setopt(__LINE__, CURLOPT_INFILESIZE, (int) $_SERVER["CONTENT_LENGTH"]);
	ch_setopt(__LINE__, CURLOPT_READFUNCTION, function($ignored1, $ignored2, $length) {
		global $preread;
		if (strlen($preread) == 0) return read_in(__LINE__, $length);
		$data = $preread . read_in(__LINE__, $length - 1);
		$preread = "";
		return $data;
	});
}


$remove_response_header = array_merge([], $hop_by_hop_header);
$resp_status_line = "";
$resp_header = [];
ch_setopt(__LINE__, CURLOPT_HEADERFUNCTION, function($ignored, $header) {
	global $remove_response_header, $resp_status_line, $resp_header;
	$lower_header = strtolower($header);
	if (strpos($lower_header, "http/") === 0) $resp_status_line = $header;
	else $resp_header[] = $header;
	if (strpos($lower_header, "connection:") === 0) foreach (explode(",", substr($lower_header, 11)) as $key) {
		$key = trim($key);
		if (strlen($key) > 0) $remove_response_header[$key] = true;
	}
	return strlen($header);
});
function send_header() {
	global $remove_response_header, $resp_status_line, $resp_header;
	if ($resp_status_line) header($resp_status_line);
	foreach ($resp_header as $header) {
		if (isset($remove_response_header[strtolower(explode(":", $header, 2)[0])])) continue;
		header($header, false);
	}
}

$data_sent = false;
ch_setopt(__LINE__, CURLOPT_WRITEFUNCTION, function($ignored, $data) {
	global $data_sent, $out;
	if (!$data_sent) {
		send_header();
		$data_sent = true;
	}
	return fwrite($out, $data);
});

if (!curl_exec($ch)) die_bad_gateway(__LINE__ . ": " . curl_strerror(curl_errno($ch)));

if (!$data_sent) send_header();

curl_close($ch);
fclose($out);
fclose($in);
