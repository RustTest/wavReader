
import {check} from 'k6';
import wavreader from 'github.com/RustTest/wavReader';

let client = wavreader.createClient({url: `pulsar://${__ENV.PULSAR_ADDR}`})
let producer = wavreader.createProducer(client, {topic: __ENV.PULSAR_TOPIC})
const audioFileLocation ="/Users/prasadchandrasekaran/code/lasthope/wavReader/652-130726-combined.wav";
let audiomessage = []; 
export default function() {
  // 3. VU code
  for(let i=0;i<audiomessage.length; i++) {
  let err = wavreader.publish(producer, audiomessage[i], {}, false);
  check(err, {
    "is send": err => err == null
  })
  }
}

export function setup() {
audiomessage = wavreader.wavReaderVoxflo(audioFileLocation,333);
client = wavreader.createClient({url: `pulsar://${__ENV.PULSAR_ADDR}`})
producer = wavreader.createProducer(client, {topic: __ENV.PULSAR_TOPIC})
}

export function teardown(data) {
  // 4. teardown code
  wavreader.closeClient(client)
  wavreader.closeProducer(producer)
  console.log("teardown!!")
}
